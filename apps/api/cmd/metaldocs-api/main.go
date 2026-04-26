package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"

	auditdomain "metaldocs/internal/modules/audit/domain"
	documents_v2 "metaldocs/internal/modules/documents_v2"
	approvalapp "metaldocs/internal/modules/documents_v2/approval/application"
	approvalhttp "metaldocs/internal/modules/documents_v2/approval/http"
	approvalrepo "metaldocs/internal/modules/documents_v2/approval/repository"
	docapp "metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/jobs"
	docrepo "metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/modules/jobs/effective_date_publisher"
	"metaldocs/internal/modules/jobs/idempotency_janitor"
	jobscheduler "metaldocs/internal/modules/jobs/scheduler"
	"metaldocs/internal/modules/jobs/stuck_instance_watchdog"
	tv2app "metaldocs/internal/modules/templates_v2/application"
	tv2http "metaldocs/internal/modules/templates_v2/delivery/http"
	tv2repo "metaldocs/internal/modules/templates_v2/repository"

	auditapp "metaldocs/internal/modules/audit/application"
	auditdelivery "metaldocs/internal/modules/audit/delivery/http"
	authapp "metaldocs/internal/modules/auth/application"
	authdelivery "metaldocs/internal/modules/auth/delivery/http"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	iamdomain "metaldocs/internal/modules/iam/domain"
	iampg "metaldocs/internal/modules/iam/infrastructure/postgres"
	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationdelivery "metaldocs/internal/modules/notifications/delivery/http"
	"metaldocs/internal/modules/registry"
	registrydomain "metaldocs/internal/modules/registry/domain"
	registryinfra "metaldocs/internal/modules/registry/infrastructure"
	"metaldocs/internal/modules/render/fanout"
	"metaldocs/internal/modules/render/resolvers"
	searchapp "metaldocs/internal/modules/search/application"
	searchdelivery "metaldocs/internal/modules/search/delivery/http"
	searchdocs "metaldocs/internal/modules/search/infrastructure/documents"
	"metaldocs/internal/modules/taxonomy"
	taxonomyinfra "metaldocs/internal/modules/taxonomy/infrastructure"
	taxonomydomain "metaldocs/internal/modules/taxonomy/domain"
	workflowapp "metaldocs/internal/modules/workflow/application"
	workflowdelivery "metaldocs/internal/modules/workflow/delivery/http"
	"metaldocs/internal/platform/authn"
	"metaldocs/internal/platform/bootstrap"
	"metaldocs/internal/platform/config"
	docgenv2 "metaldocs/internal/platform/docgenv2"
	"metaldocs/internal/platform/featureflags"
	"metaldocs/internal/platform/formval"
	"metaldocs/internal/platform/objectstore"
	"metaldocs/internal/platform/observability"
	"metaldocs/internal/platform/security"
	e2etest "metaldocs/internal/test"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repoMode, err := config.RepositoryMode()
	if err != nil {
		log.Fatalf("invalid repository mode: %v", err)
	}
	rateCfg, err := config.LoadRateLimitConfig()
	if err != nil {
		log.Fatalf("invalid rate limit config: %v", err)
	}
	corsCfg, err := config.LoadCORSConfig()
	if err != nil {
		log.Fatalf("invalid cors config: %v", err)
	}
	attachmentsCfg, err := config.LoadAttachmentsConfig()
	if err != nil {
		log.Fatalf("invalid attachments config: %v", err)
	}
	authCfg, err := authn.LoadRuntimeConfig()
	if err != nil {
		log.Fatalf("invalid auth config: %v", err)
	}
	featureFlagsCfg := config.LoadFeatureFlagsConfig()

	deps, err := bootstrap.BuildAPIDependencies(ctx, repoMode, attachmentsCfg)
	if err != nil {
		log.Fatalf("build api dependencies: %v", err)
	}
	defer deps.Cleanup()

	authService := authapp.NewService(deps.AuthRepo, deps.RoleProvider, deps.RoleAdminRepo, authCfg)
	if err := authService.BootstrapLocalAdmin(ctx); err != nil {
		log.Fatalf("bootstrap local admin: %v", err)
	}

	auditService := auditapp.NewService(deps.AuditReader)

	auditHandler := auditdelivery.NewHandler(auditService)
	searchService := searchapp.NewService(searchdocs.NewReader(deps.DocumentsRepo))
	searchHandler := searchdelivery.NewHandler(searchService)
	notificationService := notificationapp.NewService(deps.NotificationsRepo, deps.DocumentsRepo, nil)
	notificationHandler := notificationdelivery.NewHandler(notificationService)
	workflowService := workflowapp.NewService(deps.DocumentsRepo, deps.WorkflowApprovals, deps.AuditWriter, deps.Publisher, nil)
	workflowHandler := workflowdelivery.NewHandler(workflowService)
	authHandler := authdelivery.NewHandler(authService)
	healthHandler := observability.NewHealthHandler(deps.StatusProvider)

	authorizer := iamapp.NewStaticAuthorizer()
	cachedProvider := iamapp.NewCachedRoleProvider(deps.RoleProvider, authn.CacheTTL())
	// permResolver is the single authoritative source of truth for route
	// visibility. It is shared with the auth middleware so that fully public
	// routes (no session required) and the IAM permission layer stay in sync
	// automatically — adding a new public route requires one change here, not two.
	permResolver := newPermissionResolver()
	authMiddleware := authdelivery.NewMiddleware(authService, authCfg, authn.Enabled()).
		WithPublicPathChecker(newPublicPathChecker(permResolver))
	iamMiddleware := iamdelivery.NewMiddleware(authorizer, cachedProvider, authn.Enabled(), authCfg.LegacyHeaderEnabled).
		WithPermissionResolver(permResolver)
	originProtection := security.NewOriginProtection(security.OriginProtectionConfig{
		Enabled:           authCfg.OriginProtection,
		SessionCookieName: authCfg.SessionCookieName,
		TrustedOrigins:    authCfg.TrustedOrigins,
	})

	iamAdminService := iamapp.NewAdminService(deps.RoleAdminRepo, cachedProvider)
	iamAdminHandler := iamdelivery.NewAdminHandler(iamAdminService, authService, deps.AuditWriter).
		WithAuditReader(deps.AuditReader)
	featureFlagsHandler := featureflags.NewHandler(featureFlagsCfg)
	httpObs := observability.NewHTTPObservability(deps.StatusProvider)
	rateLimiter := security.NewRateLimiter(rateCfg)
	cors := security.NewCORS(corsCfg)

	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	healthHandler.RegisterRoutes(mux)
	featureFlagsHandler.RegisterRoutes(mux)
	auditHandler.RegisterRoutes(mux)
	searchHandler.RegisterRoutes(mux)
	notificationHandler.RegisterRoutes(mux)
	workflowHandler.RegisterRoutes(mux)
	iamAdminHandler.RegisterRoutes(mux)

	taxonomyModule := taxonomy.New(taxonomy.Dependencies{
		DB:         deps.SQLDB,
		TplChecker: taxonomyinfra.NewTemplateVersionChecker(deps.SQLDB),
	})
	taxonomyModule.RegisterRoutes(mux)

	registryModule := registry.New(registry.Dependencies{
		DB:     deps.SQLDB,
		Logger: slog.Default(),
	})
	registryModule.RegisterRoutes(mux)
	if deps.SQLDB != nil {
		if err := registryModule.RunStartupMigrations(context.Background(), deps.SQLDB, slog.Default()); err != nil {
			log.Printf("registry startup migration failed: %v", err)
		}
	}

	var membershipService *iamapp.AreaMembershipService
	if deps.SQLDB != nil {
		membershipService = iamapp.NewAreaMembershipService(iampg.NewUserAreaRepository(deps.SQLDB), nil)
	}
	iamdelivery.NewMembershipHandler(membershipService).RegisterRoutes(mux)

	// Legacy templates module routes removed — templates_v2 owns /api/v2/templates/*

	docPresigner := objectstore.NewDocumentPresigner(deps.MinioClient, deps.MinioBucket, 15*time.Minute, 25*1024*1024)
	cdRepo := registryinfra.NewPostgresControlledDocumentRepository(deps.SQLDB)
	profileRepo := taxonomyinfra.NewProfileRepository(deps.SQLDB)

	// Fanout/eigenpal client — enabled when METALDOCS_FANOUT_URL is set.
	fanoutURL := strings.TrimSpace(os.Getenv("METALDOCS_FANOUT_URL"))
	if fanoutURL == "" {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("METALDOCS_REQUIRE_FANOUT")), "true") {
			log.Fatalf("METALDOCS_FANOUT_URL is required but not set")
		}
		slog.Warn("METALDOCS_FANOUT_URL not set; document approval will fail at freeze step")
	}
	var fanoutCli *fanout.Client
	var freezeSvc *docapp.FreezeService
	if fanoutURL != "" && deps.SQLDB != nil {
		fanoutCli = fanout.NewClient(fanoutURL, nil)
		snapRepo := docrepo.NewSnapshotRepository(deps.SQLDB)
		fillInRepo := docrepo.NewFillInRepository(deps.SQLDB)
		schemaReader := docapp.NewSnapshotSchemaReader(deps.SQLDB)
		revReader := docrepo.NewRevisionReader(deps.SQLDB)
		wfReader := docrepo.NewWorkflowReader(deps.SQLDB)
		ctxBuilder := docapp.NewDocumentContextBuilder(deps.SQLDB, revReader, wfReader,
			cdRegistryAdapter{cdRepo})
		resolverReg := resolvers.NewRegistry()
		resolvers.RegisterBuiltins(resolverReg)
		freezeSvc = docapp.NewFreezeService(
			schemaReader, fillInRepo, fillInRepo,
			resolverReg, snapRepo, ctxBuilder,
			snapRepo, snapRepo, fillInRepo, fanoutCli,
		)
	}

	docDeps := documents_v2.Dependencies{
		DB:      deps.SQLDB,
		Docgen:  nil,
		Presign: docPresigner,
		TplRead: docgenv2.NewFanoutTemplateReader(
			docgenv2.NewTemplateReader(deps.SQLDB, deps.MinioClient, deps.MinioBucket),
			docgenv2.NewTemplatesV2TemplateReader(deps.SQLDB),
		),
		FormVal:         formval.NewGojsonschema(),
		Audit:           newDocumentsV2AuditAdapter(deps.AuditWriter),
		ExportPresign:   docPresigner,
		RegistryReader:  cdRepo,
		AuthzChecker:    permissiveAuthzChecker{},
		ProfileDefaults: &profileDefaultsAdapter{profileRepo: profileRepo},
	}
	if deps.DocgenV2Client != nil {
		docDeps.ExportDocgen = deps.DocgenV2Client
	}
	if fanoutCli != nil && deps.SQLDB != nil {
		snapRepo := docrepo.NewSnapshotRepository(deps.SQLDB)
		inputsReader := docrepo.NewFanoutInputsReader(deps.SQLDB)
		docDeps.ReconstructRunner = fanout.NewReconstructService(
			inputsReader, fanoutCli, snapRepo,
			fanout.EngineVersions{EigenpalVer: "local", DocxtemplaterVer: "local"},
			nil,
		)
	}
	docMod := documents_v2.New(docDeps)
	docMod.RegisterRoutes(mux)

	tv2Presigner := objectstore.NewTemplatesV2Presigner(deps.MinioClient, deps.MinioBucket, 25*1024*1024)
	tv2Svc := tv2app.New(tv2repo.New(deps.SQLDB), tv2Presigner, realClock{}, realUUIDGen{})
	tv2http.New(tv2Svc, nil).Register(mux)

	approvalRepo := approvalrepo.NewPostgresApprovalRepository(deps.SQLDB)
	approvalEmitter := approvalapp.NewSQLEmitter()
	approvalServices := approvalapp.NewServices(approvalRepo, approvalEmitter, approvalapp.RealClock{})
	if freezeSvc != nil {
		approvalServices.Decision = approvalapp.NewDecisionService(
			approvalRepo, approvalEmitter, approvalapp.RealClock{}, freezeSvc, nil,
		)
	}
	approvalHandler := approvalhttp.NewHandler(approvalServices, deps.SQLDB)
	approvalHandler.RegisterRoutes(mux)
	e2etest.RegisterE2EHandlers(mux, deps.SQLDB, func(ctx context.Context) error {
		_, err := approvalServices.Scheduler.RunDuePublishes(ctx, deps.SQLDB)
		return err
	})

	leaderID := schedulerLeaderID()
	s := jobscheduler.New(deps.SQLDB, leaderID)
	if jobEnabled("ENABLE_JOB_EFFECTIVE_DATE_PUBLISHER") {
		s.Register(jobscheduler.JobConfig{
			Name:     "effective-date-publisher",
			Interval: time.Minute,
			Fn:       effective_date_publisher.New(deps.SQLDB, approvalServices.Scheduler),
			Policy:   jobscheduler.SkipOnPressure,
		})
	}
	if jobEnabled("ENABLE_JOB_STUCK_INSTANCE_WATCHDOG") {
		s.Register(jobscheduler.JobConfig{
			Name:     "stuck-instance-watchdog",
			Interval: 5 * time.Minute,
			Fn:       stuck_instance_watchdog.New(deps.SQLDB, approvalServices.Cancel, approvalEmitter),
			Policy:   jobscheduler.SkipOnPressure,
		})
	}
	if jobEnabled("ENABLE_JOB_IDEMPOTENCY_JANITOR") {
		s.Register(jobscheduler.JobConfig{
			Name:     "idempotency-janitor",
			Interval: 15 * time.Minute,
			Fn:       idempotency_janitor.New(deps.SQLDB),
			Policy:   jobscheduler.SkipOnPressure,
		})
	}
	if jobEnabled("ENABLE_JOB_LEASE_REAPER") {
		s.Register(jobscheduler.JobConfig{
			Name:     "lease-reaper",
			Interval: 10 * time.Minute,
			Fn:       jobscheduler.RunLeaseReaper(deps.SQLDB),
			Policy:   jobscheduler.SkipOnPressure,
		})
	}

	var schedulerWG sync.WaitGroup
	schedulerWG.Add(1)
	go func() {
		defer schedulerWG.Done()
		s.Start(ctx)
	}()

	stopSessions := jobs.StartSessionSweeper(ctx, docMod.Repo(), 60*time.Second)
	stopOrphans := jobs.StartOrphanPendingSweeper(ctx, docMod.Repo(), time.Hour)
	defer stopSessions()
	defer stopOrphans()
	mux.Handle("/api/v1/metrics", httpObs.MetricsHandler())

	handler := cors.Wrap(originProtection.Wrap(authMiddleware.Wrap(iamMiddleware.Wrap(httpObs.Wrap(rateLimiter.Wrap(mux))))))

	addr := ":8080"
	if appPort := os.Getenv("APP_PORT"); appPort != "" {
		port, convErr := strconv.Atoi(strings.TrimSpace(appPort))
		if convErr != nil || port < 1 || port > 65535 {
			log.Fatalf("invalid APP_PORT value")
		}
		addr = ":" + strconv.Itoa(port)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("MetalDocs API listening on %s (repository=%s auth_enabled=%t auth_cache_ttl=%s rate_limit_enabled=%t rate_limit_window_s=%d rate_limit_max_requests=%d cors_enabled=%t cors_allowed_origins=%d)",
		addr, repoMode, authn.Enabled(), authn.CacheTTL(), rateCfg.Enabled, rateCfg.WindowSeconds, rateCfg.MaxRequests, corsCfg.Enabled, len(corsCfg.AllowedOrigins))

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			stop()
			schedulerWG.Wait()
			log.Fatalf("server failed: %v", err)
		}
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	stop()
	schedulerWG.Wait()
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type realUUIDGen struct{}

func (realUUIDGen) New() string { return uuid.NewString() }

type documentsV2AuditAdapter struct {
	writer auditdomain.Writer
}

func newDocumentsV2AuditAdapter(writer auditdomain.Writer) *documentsV2AuditAdapter {
	return &documentsV2AuditAdapter{writer: writer}
}

func (a *documentsV2AuditAdapter) Write(ctx context.Context, tenantID, actorID, action, docID string, meta any) {
	if a == nil || a.writer == nil {
		return
	}

	payload := map[string]any{"tenant_id": tenantID}
	if meta != nil {
		payload["meta"] = meta
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte("{}")
	}

	if err := a.writer.Record(ctx, auditdomain.Event{
		ID:           uuid.NewString(),
		OccurredAt:   time.Now().UTC(),
		ActorID:      actorID,
		Action:       action,
		ResourceType: "document",
		ResourceID:   docID,
		PayloadJSON:  string(raw),
		TraceID:      "trace-local",
	}); err != nil {
		log.Printf("documents_v2 audit write failed: %v", err)
	}
}

func jobEnabled(envName string) bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv(envName)), "false")
}

func schedulerLeaderID() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s:%d", hostname, os.Getpid())
}

// permissiveAuthzChecker always grants access (dev/MVP only — IAM area check not yet enforced).
type permissiveAuthzChecker struct{}

func (permissiveAuthzChecker) Check(_ context.Context, _, _ string, _ iamdomain.Capability, _ iamapp.ResourceCtx) error {
	return nil
}

// cdRegistryAdapter bridges the registry ControlledDocumentRepository → resolvers.RegistryReader.
type cdRegistryAdapter struct {
	repo interface {
		GetByID(ctx context.Context, tenantID, id string) (*registrydomain.ControlledDocument, error)
	}
}

func (a cdRegistryAdapter) GetControlledDocument(ctx context.Context, tenantID, controlledDocumentID string) (resolvers.ControlledDocumentInfo, error) {
	cd, err := a.repo.GetByID(ctx, tenantID, controlledDocumentID)
	if err != nil {
		return resolvers.ControlledDocumentInfo{}, err
	}
	return resolvers.ControlledDocumentInfo{DocCode: cd.Code}, nil
}

// profileDefaultsAdapter bridges taxonomy ProfileRepository → documents_v2 ProfileDefaultTemplateReader.
type profileDefaultsAdapter struct {
	profileRepo interface {
		GetByCode(ctx context.Context, tenantID, code string) (*taxonomydomain.DocumentProfile, error)
	}
}

func (a *profileDefaultsAdapter) GetDefaultTemplateVersionID(ctx context.Context, tenantID, profileCode string) (*string, *string, error) {
	profile, err := a.profileRepo.GetByCode(ctx, tenantID, profileCode)
	if err != nil {
		return nil, nil, err
	}
	if profile.DefaultTemplateVersionID == nil {
		return nil, nil, nil
	}
	status := "published"
	return profile.DefaultTemplateVersionID, &status, nil
}

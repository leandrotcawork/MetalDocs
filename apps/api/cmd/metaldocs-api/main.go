package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "metaldocs/internal/modules/document_revisions"
	_ "metaldocs/internal/modules/editor_sessions"

	auditdomain "metaldocs/internal/modules/audit/domain"
	documents_v2 "metaldocs/internal/modules/documents_v2"
	"metaldocs/internal/modules/documents_v2/jobs"
	templatesmod "metaldocs/internal/modules/templates"

	auditapp "metaldocs/internal/modules/audit/application"
	auditdelivery "metaldocs/internal/modules/audit/delivery/http"
	authapp "metaldocs/internal/modules/auth/application"
	authdelivery "metaldocs/internal/modules/auth/delivery/http"
	docapp "metaldocs/internal/modules/documents/application"
	docdelivery "metaldocs/internal/modules/documents/delivery/http"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationdelivery "metaldocs/internal/modules/notifications/delivery/http"
	searchapp "metaldocs/internal/modules/search/application"
	searchdelivery "metaldocs/internal/modules/search/delivery/http"
	searchdocs "metaldocs/internal/modules/search/infrastructure/documents"
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
)

func main() {
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

	deps, err := bootstrap.BuildAPIDependencies(context.Background(), repoMode, attachmentsCfg)
	if err != nil {
		log.Fatalf("build api dependencies: %v", err)
	}
	defer deps.Cleanup()

	authService := authapp.NewService(deps.AuthRepo, deps.RoleProvider, deps.RoleAdminRepo, authCfg)
	if err := authService.BootstrapLocalAdmin(context.Background()); err != nil {
		log.Fatalf("bootstrap local admin: %v", err)
	}

	auditService := auditapp.NewService(deps.AuditReader)
	docService := docapp.NewService(deps.DocumentsRepo, deps.Publisher, nil).
		WithAttachmentStore(deps.AttachmentStore).
		WithAuditWriter(deps.AuditWriter).
		WithGotenberg(deps.GotenbergClient).
		WithApprovalReader(docapp.NewWorkflowApprovalAdapter(deps.WorkflowApprovals))

	auditHandler := auditdelivery.NewHandler(auditService)
	docHandler := docdelivery.NewHandler(docService).
		WithAttachmentDownloads(security.NewAttachmentSigner(attachmentsCfg.DownloadSecret), time.Duration(attachmentsCfg.DownloadTTLSeconds)*time.Second).
		WithPDFConverter(deps.GotenbergClient)
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
	docHandler.RegisterRoutes(mux)
	searchHandler.RegisterRoutes(mux)
	notificationHandler.RegisterRoutes(mux)
	workflowHandler.RegisterRoutes(mux)
	iamAdminHandler.RegisterRoutes(mux)
	if featureFlagsCfg.DocxV2Enabled {
		// TODO(Task 22): wire real presigner once APIDependencies exposes S3Client + S3Bucket.
		// presigner := objectstore.NewTemplatePresigner(deps.S3Client, deps.S3Bucket, 15*time.Minute, 10*1024*1024)
		tplMod := templatesmod.New(deps.SQLDB, deps.DocgenV2Client, nil)
		tplMod.RegisterRoutes(mux)
		log.Printf("docx-v2 templates module enabled")

		// TODO(docx-v2): APIDependencies currently does not expose S3Client/S3Bucket.
		// Pass nil placeholders until bootstrap wiring is added.
		docMod := documents_v2.New(documents_v2.Dependencies{
			DB:      deps.SQLDB,
			Docgen:  nil, // TODO(docx-v2): adapt DocgenV2Client to application.DocgenRenderer
			Presign: objectstore.NewDocumentPresigner(nil, "", 15*time.Minute, 25*1024*1024),
			TplRead: docgenv2.NewTemplateReader(deps.SQLDB, nil, ""),
			FormVal: formval.NewGojsonschema(),
			Audit:   newDocumentsV2AuditAdapter(deps.AuditWriter),
		})
		docMod.RegisterRoutes(mux)
		log.Printf("docx-v2 documents module enabled")

		stopSessions := jobs.StartSessionSweeper(context.Background(), docMod.Repo(), 60*time.Second)
		stopOrphans := jobs.StartOrphanPendingSweeper(context.Background(), docMod.Repo(), time.Hour)
		defer stopSessions()
		defer stopOrphans()
	}
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
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

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

	_ = a.writer.Record(ctx, auditdomain.Event{
		ActorID:      actorID,
		Action:       action,
		ResourceType: "document",
		ResourceID:   docID,
		PayloadJSON:  string(raw),
	})
}

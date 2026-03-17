package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	auditdomain "metaldocs/internal/modules/audit/domain"
	auditmemory "metaldocs/internal/modules/audit/infrastructure/memory"
	auditpg "metaldocs/internal/modules/audit/infrastructure/postgres"
	docapp "metaldocs/internal/modules/documents/application"
	docdelivery "metaldocs/internal/modules/documents/delivery/http"
	docdomain "metaldocs/internal/modules/documents/domain"
	memoryrepo "metaldocs/internal/modules/documents/infrastructure/memory"
	pgrepo "metaldocs/internal/modules/documents/infrastructure/postgres"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	iamdomain "metaldocs/internal/modules/iam/domain"
	iammemory "metaldocs/internal/modules/iam/infrastructure/memory"
	iampg "metaldocs/internal/modules/iam/infrastructure/postgres"
	searchapp "metaldocs/internal/modules/search/application"
	searchdelivery "metaldocs/internal/modules/search/delivery/http"
	searchdocs "metaldocs/internal/modules/search/infrastructure/documents"
	workflowapp "metaldocs/internal/modules/workflow/application"
	workflowdelivery "metaldocs/internal/modules/workflow/delivery/http"
	"metaldocs/internal/platform/authn"
	"metaldocs/internal/platform/config"
	pgdb "metaldocs/internal/platform/db/postgres"
	"metaldocs/internal/platform/messaging"
	nooppub "metaldocs/internal/platform/messaging/noop"
	outboxpg "metaldocs/internal/platform/messaging/outbox/postgres"
	"metaldocs/internal/platform/observability"
	"metaldocs/internal/platform/security"
	localstorage "metaldocs/internal/platform/storage/local"
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
	attachmentsCfg, err := config.LoadAttachmentsConfig()
	if err != nil {
		log.Fatalf("invalid attachments config: %v", err)
	}

	docRepo, attachmentStore, roleProvider, roleAdminRepo, auditWriter, publisher, cleanup := buildDependencies(repoMode, attachmentsCfg)
	defer cleanup()

	docService := docapp.NewService(docRepo, publisher, nil).WithAttachmentStore(attachmentStore)
	docHandler := docdelivery.NewHandler(docService).WithAttachmentDownloads(security.NewAttachmentSigner(attachmentsCfg.DownloadSecret), time.Duration(attachmentsCfg.DownloadTTLSeconds)*time.Second)
	searchService := searchapp.NewService(searchdocs.NewReader(docRepo))
	searchHandler := searchdelivery.NewHandler(searchService)
	workflowService := workflowapp.NewService(docRepo, auditWriter, publisher, nil)
	workflowHandler := workflowdelivery.NewHandler(workflowService)

	authorizer := iamapp.NewStaticAuthorizer()
	cachedProvider := iamapp.NewCachedRoleProvider(roleProvider, authn.CacheTTL())
	iamMiddleware := iamdelivery.NewMiddleware(authorizer, cachedProvider, authn.Enabled())

	iamAdminService := iamapp.NewAdminService(roleAdminRepo, cachedProvider)
	iamAdminHandler := iamdelivery.NewAdminHandler(iamAdminService)
	httpObs := observability.NewHTTPObservability()
	rateLimiter := security.NewRateLimiter(rateCfg)

	mux := http.NewServeMux()
	docHandler.RegisterRoutes(mux)
	searchHandler.RegisterRoutes(mux)
	workflowHandler.RegisterRoutes(mux)
	iamAdminHandler.RegisterRoutes(mux)
	mux.Handle("/api/v1/metrics", httpObs.MetricsHandler())

	handler := httpObs.Wrap(rateLimiter.Wrap(iamMiddleware.Wrap(mux)))

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

	log.Printf("MetalDocs API listening on %s (repository=%s auth_enabled=%t auth_cache_ttl=%s rate_limit_enabled=%t rate_limit_window_s=%d rate_limit_max_requests=%d)",
		addr, repoMode, authn.Enabled(), authn.CacheTTL(), rateCfg.Enabled, rateCfg.WindowSeconds, rateCfg.MaxRequests)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func buildDependencies(mode string, attachmentsCfg config.AttachmentsConfig) (docdomain.Repository, docdomain.AttachmentStore, iamdomain.RoleProvider, iamdomain.RoleAdminRepository, auditdomain.Writer, messaging.Publisher, func()) {
	switch mode {
	case config.RepositoryPostgres:
		pgCfg, err := config.LoadPostgresConfig()
		if err != nil {
			log.Fatalf("load postgres config: %v", err)
		}
		db, err := pgdb.Open(context.Background(), pgCfg.DSN)
		if err != nil {
			log.Fatalf("open postgres: %v", err)
		}
		return pgrepo.NewRepository(db), localstorage.NewStore(attachmentsCfg.RootPath), iampg.NewRoleProvider(db), iampg.NewRoleAdminRepository(db), auditpg.NewWriter(db), outboxpg.NewPublisher(db), func() { _ = closeDB(db) }
	default:
		roles := authn.DevRoleMap()
		return memoryrepo.NewRepository(), memoryrepo.NewAttachmentStore(), iamapp.NewDevRoleProvider(roles), iammemory.NewRoleAdminRepository(), auditmemory.NewWriter(), nooppub.NewPublisher(), func() {}
	}
}

func closeDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

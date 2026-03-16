package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

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
)

func main() {
	repoMode, err := config.RepositoryMode()
	if err != nil {
		log.Fatalf("invalid repository mode: %v", err)
	}

	docRepo, roleProvider, roleAdminRepo, auditWriter, cleanup := buildDependencies(repoMode)
	defer cleanup()

	docService := docapp.NewService(docRepo, nil, nil)
	docHandler := docdelivery.NewHandler(docService)
	searchService := searchapp.NewService(searchdocs.NewReader(docRepo))
	searchHandler := searchdelivery.NewHandler(searchService)
	workflowService := workflowapp.NewService(docRepo, auditWriter, nil, nil)
	workflowHandler := workflowdelivery.NewHandler(workflowService)

	authorizer := iamapp.NewStaticAuthorizer()
	cachedProvider := iamapp.NewCachedRoleProvider(roleProvider, authn.CacheTTL())
	iamMiddleware := iamdelivery.NewMiddleware(authorizer, cachedProvider, authn.Enabled())

	iamAdminService := iamapp.NewAdminService(roleAdminRepo, cachedProvider)
	iamAdminHandler := iamdelivery.NewAdminHandler(iamAdminService)

	mux := http.NewServeMux()
	docHandler.RegisterRoutes(mux)
	searchHandler.RegisterRoutes(mux)
	workflowHandler.RegisterRoutes(mux)
	iamAdminHandler.RegisterRoutes(mux)

	handler := iamMiddleware.Wrap(mux)

	addr := ":8080"
	if appPort := os.Getenv("APP_PORT"); appPort != "" {
		addr = ":" + appPort
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	log.Printf("MetalDocs API listening on %s (repository=%s auth_enabled=%t auth_cache_ttl=%s)", addr, repoMode, authn.Enabled(), authn.CacheTTL())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func buildDependencies(mode string) (docdomain.Repository, iamdomain.RoleProvider, iamdomain.RoleAdminRepository, auditdomain.Writer, func()) {
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
		return pgrepo.NewRepository(db), iampg.NewRoleProvider(db), iampg.NewRoleAdminRepository(db), auditpg.NewWriter(db), func() { _ = closeDB(db) }
	default:
		roles := authn.DevRoleMap()
		return memoryrepo.NewRepository(), iamapp.NewDevRoleProvider(roles), iammemory.NewRoleAdminRepository(), auditmemory.NewWriter(), func() {}
	}
}

func closeDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

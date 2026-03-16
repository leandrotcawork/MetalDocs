package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	docapp "metaldocs/internal/modules/documents/application"
	docdelivery "metaldocs/internal/modules/documents/delivery/http"
	docdomain "metaldocs/internal/modules/documents/domain"
	memoryrepo "metaldocs/internal/modules/documents/infrastructure/memory"
	pgrepo "metaldocs/internal/modules/documents/infrastructure/postgres"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	iamdomain "metaldocs/internal/modules/iam/domain"
	iampg "metaldocs/internal/modules/iam/infrastructure/postgres"
	"metaldocs/internal/platform/authn"
	"metaldocs/internal/platform/config"
	pgdb "metaldocs/internal/platform/db/postgres"
)

func main() {
	repoMode, err := config.RepositoryMode()
	if err != nil {
		log.Fatalf("invalid repository mode: %v", err)
	}

	docRepo, roleProvider, cleanup := buildDependencies(repoMode)
	defer cleanup()

	docService := docapp.NewService(docRepo, nil, nil)
	docHandler := docdelivery.NewHandler(docService)

	mux := http.NewServeMux()
	docHandler.RegisterRoutes(mux)

	authorizer := iamapp.NewStaticAuthorizer()
	cachedProvider := iamapp.NewCachedRoleProvider(roleProvider, authn.CacheTTL())
	iamMiddleware := iamdelivery.NewMiddleware(authorizer, cachedProvider, authn.Enabled())
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

func buildDependencies(mode string) (docdomain.Repository, iamdomain.RoleProvider, func()) {
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
		return pgrepo.NewRepository(db), iampg.NewRoleProvider(db), func() { _ = closeDB(db) }
	default:
		roles := authn.DevRoleMap()
		return memoryrepo.NewRepository(), iamapp.NewDevRoleProvider(roles), func() {}
	}
}

func closeDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

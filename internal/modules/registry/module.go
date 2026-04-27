package registry

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"metaldocs/internal/modules/registry/application"
	dhttp "metaldocs/internal/modules/registry/delivery/http"
	"metaldocs/internal/modules/registry/infrastructure"
	taxonomyapp "metaldocs/internal/modules/taxonomy/application"
)

type Module struct {
	Handler *dhttp.Handler
	svc     *application.RegistryService
}

type Dependencies struct {
	DB     *sql.DB
	Logger *slog.Logger
}

func New(deps Dependencies) *Module {
	repo := infrastructure.NewPostgresControlledDocumentRepository(deps.DB)
	seq := infrastructure.NewPostgresSequenceAllocator(deps.DB)
	tplCheck := infrastructure.NewPostgresTemplateVersionChecker(deps.DB)
	profiles := infrastructure.NewTaxonomyProfileReader(deps.DB)
	areas := infrastructure.NewTaxonomyAreaReader(deps.DB)
	govLogger := taxonomyapp.NewDBGovernanceLogger(deps.DB)
	svc := application.NewRegistryService(deps.DB, repo, seq, tplCheck, profiles, areas, govLogger)
	h := dhttp.NewHandler(svc, deps.DB)
	return &Module{Handler: h, svc: svc}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	m.Handler.RegisterRoutes(mux)
}

func (m *Module) RunStartupMigrations(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	return application.BackfillLegacyDocuments(ctx, db, logger)
}

func (m *Module) Service() *application.RegistryService { return m.svc }

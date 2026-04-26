package documents_v2

import (
	"database/sql"
	"net/http"

	"metaldocs/internal/modules/documents_v2/application"
	dhttp "metaldocs/internal/modules/documents_v2/delivery/http"
	documentshttp "metaldocs/internal/modules/documents_v2/http"
	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/platform/ratelimit"
)

type Module struct {
	Handler            *dhttp.Handler
	ExportHandler      *dhttp.ExportHandler
	FillInHandler      *documentshttp.FillInHandler
	ViewHandler        *documentshttp.ViewHandler
	ReconstructHandler *documentshttp.ReconstructHandler
	repo               *repository.Repository
}

type Dependencies struct {
	DB              *sql.DB
	Docgen          application.DocgenRenderer
	Presign         application.Presigner
	TplRead         application.TemplateReader
	FormVal         application.FormValidator
	Audit           application.Audit
	RegistryReader  application.RegistryReader
	AuthzChecker    application.AuthorizationChecker
	ProfileDefaults application.ProfileDefaultTemplateReader
	ExportPresign   application.ExportPresigner
	ExportDocgen      application.DocgenPDFClient
	DocgenVer         string
	GrammarVer        string
	ReconstructRunner application.ReconstructionRunner
}

func New(deps Dependencies) *Module {
	repo := repository.New(deps.DB)
	svc := application.NewService(repo, deps.Docgen, deps.Presign, deps.TplRead, deps.FormVal, deps.Audit, deps.RegistryReader, deps.AuthzChecker, deps.ProfileDefaults)
	h := dhttp.NewHandler(svc)

	var exportHandler *dhttp.ExportHandler
	if deps.ExportPresign != nil && deps.ExportDocgen != nil {
		docgenVer := deps.DocgenVer
		if docgenVer == "" {
			docgenVer = "docgen-v2@0.4.0"
		}
		grammarVer := deps.GrammarVer
		if grammarVer == "" {
			grammarVer = "grammar-v1"
		}
		exportSvc := application.NewExportService(repo, deps.ExportPresign, deps.ExportDocgen, deps.Audit, docgenVer, grammarVer)
		exportHandler = dhttp.NewExportHandler(exportSvc)
	}

	fillInRepo := repository.NewFillInRepository(deps.DB)
	fillInSvc := application.NewFillInService(deps.DB, application.NewSnapshotSchemaReader(deps.DB), fillInRepo).
		WithReader(fillInRepo).
		WithTemplateSchemaReader(application.NewTemplateVersionSchemaReader(deps.DB))
	fillInHandler := documentshttp.NewFillInHandler(fillInSvc)

	var viewHandler *documentshttp.ViewHandler
	if deps.Presign != nil && deps.DB != nil {
		viewSvc := application.NewViewService(deps.DB, deps.Presign)
		viewHandler = documentshttp.NewViewHandler(viewSvc)
	}

	var reconstructHandler *documentshttp.ReconstructHandler
	if deps.ReconstructRunner != nil && deps.DB != nil {
		reconstructSvc := application.NewReconstructionService(deps.DB, deps.ReconstructRunner)
		reconstructHandler = documentshttp.NewReconstructHandler(reconstructSvc)
	}

	return &Module{
		Handler:            h,
		ExportHandler:      exportHandler,
		FillInHandler:      fillInHandler,
		ViewHandler:        viewHandler,
		ReconstructHandler: reconstructHandler,
		repo:               repo,
	}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	m.Handler.RegisterRoutes(mux)
	if m.ExportHandler != nil {
		m.ExportHandler.RegisterRoutes(mux)
	}
	m.FillInHandler.RegisterRoutes(mux)
	if m.ViewHandler != nil {
		m.ViewHandler.RegisterRoutes(mux)
	}
	if m.ReconstructHandler != nil {
		m.ReconstructHandler.RegisterRoutes(mux)
	}
}

func (m *Module) RegisterRoutesWithRateLimit(mux *http.ServeMux, rl *ratelimit.Middleware, userFn func(*http.Request) string) {
	m.Handler.RegisterRoutesWithRateLimit(mux, rl, userFn)
	if m.ExportHandler != nil {
		m.ExportHandler.RegisterRoutesWithRateLimit(mux, rl, userFn)
	}
	m.FillInHandler.RegisterRoutes(mux)
	if m.ViewHandler != nil {
		m.ViewHandler.RegisterRoutes(mux)
	}
	if m.ReconstructHandler != nil {
		m.ReconstructHandler.RegisterRoutes(mux)
	}
}

func (m *Module) Repo() *repository.Repository { return m.repo }

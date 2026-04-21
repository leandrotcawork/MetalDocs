package documents_v2

import (
	"database/sql"
	"net/http"

	"metaldocs/internal/modules/documents_v2/application"
	dhttp "metaldocs/internal/modules/documents_v2/delivery/http"
	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/platform/ratelimit"
)

type Module struct {
	Handler       *dhttp.Handler
	ExportHandler *dhttp.ExportHandler
	repo          *repository.Repository
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
	ExportDocgen    application.DocgenPDFClient
	DocgenVer       string
	GrammarVer      string
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

	return &Module{
		Handler:       h,
		ExportHandler: exportHandler,
		repo:          repo,
	}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	m.Handler.RegisterRoutes(mux)
	if m.ExportHandler != nil {
		m.ExportHandler.RegisterRoutes(mux)
	}
}

func (m *Module) RegisterRoutesWithRateLimit(mux *http.ServeMux, rl *ratelimit.Middleware, userFn func(*http.Request) string) {
	m.Handler.RegisterRoutesWithRateLimit(mux, rl, userFn)
	if m.ExportHandler != nil {
		m.ExportHandler.RegisterRoutesWithRateLimit(mux, rl, userFn)
	}
}

func (m *Module) Repo() *repository.Repository { return m.repo }

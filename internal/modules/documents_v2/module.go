package documents_v2

import (
	"database/sql"
	"net/http"

	"metaldocs/internal/modules/documents_v2/application"
	dhttp "metaldocs/internal/modules/documents_v2/delivery/http"
	"metaldocs/internal/modules/documents_v2/repository"
)

type Module struct {
	Handler *dhttp.Handler
	repo    *repository.Repository
}

type Dependencies struct {
	DB      *sql.DB
	Docgen  application.DocgenRenderer
	Presign application.Presigner
	TplRead application.TemplateReader
	FormVal application.FormValidator
	Audit   application.Audit
}

func New(deps Dependencies) *Module {
	repo := repository.New(deps.DB)
	svc := application.New(repo, deps.Docgen, deps.Presign, deps.TplRead, deps.FormVal, deps.Audit)
	return &Module{Handler: dhttp.NewHandler(svc), repo: repo}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) { m.Handler.RegisterRoutes(mux) }

func (m *Module) Repo() *repository.Repository { return m.repo }

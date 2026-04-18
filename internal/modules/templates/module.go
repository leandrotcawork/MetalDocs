package templates

import (
	"database/sql"
	"net/http"

	"metaldocs/internal/modules/templates/application"
	thttp "metaldocs/internal/modules/templates/delivery/http"
	"metaldocs/internal/modules/templates/repository"
)

type Module struct {
	Handler *thttp.Handler
}

func New(db *sql.DB, docgen application.DocgenValidator, presigner application.Presigner) *Module {
	repo := repository.New(db)
	svc := application.New(repo, docgen, presigner)
	return &Module{Handler: thttp.NewHandler(svc)}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	m.Handler.RegisterRoutes(mux)
}

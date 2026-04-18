package ratelimit

// Per-route quotas from spec §Rate limits. Values are requests-per-minute
// per user. Routes not listed here are unlimited.
//
// Envvar overrides: METALDOCS_RLIMIT_<ROUTE_KEY> (e.g. EXPORT_PDF=30).
type RouteKey string

const (
    RouteUploadsPresign   RouteKey = "uploads_presign"
    RouteAutosavePresign  RouteKey = "autosave_presign"
    RouteAutosaveCommit   RouteKey = "autosave_commit"
    RouteDocumentsRender  RouteKey = "documents_render"
    RouteExportPDF        RouteKey = "export_pdf"
)

type Config struct {
    Quotas map[RouteKey]int // req/min
}

func DefaultConfig() Config {
    return Config{
        Quotas: map[RouteKey]int{
            RouteUploadsPresign:  60,
            RouteAutosavePresign: 60,
            RouteAutosaveCommit:  30,
            RouteDocumentsRender: 30,
            RouteExportPDF:       20,
        },
    }
}

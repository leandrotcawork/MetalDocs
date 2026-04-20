import type { WorkspaceView } from "../components/DocumentWorkspaceShell";

type DocumentsScopeView = "library" | "my-docs" | "recent";

const documentsBaseByView: Record<DocumentsScopeView, string> = {
  "library": "/documents",
  "my-docs": "/documents/mine",
  "recent": "/documents/recent",
};

function normalizePath(pathname: string): string {
  if (!pathname) return "/";
  if (pathname.length > 1 && pathname.endsWith("/")) {
    return pathname.slice(0, -1);
  }
  return pathname;
}

export function documentsBasePath(view: DocumentsScopeView): string {
  return documentsBaseByView[view];
}

type DocumentsRoute =
  | { view: "overview" }
  | { view: "collection"; areaCode?: string; profileCode?: string }
  | { view: "detail"; documentId: string };

export function viewFromPath(pathname: string): WorkspaceView {
  const path = normalizePath(pathname);

  if (path === "/documents-v2/new" || path === "/documents-v2") return "documents-v2";
  if (path.startsWith("/documents-v2/")) return "documents-v2";
  if (path.startsWith("/documents/mine")) return "my-docs";
  if (path.startsWith("/documents/recent")) return "recent";
  if (path.startsWith("/documents")) return "library";
  if (path.startsWith("/create")) return "create";
  if (path.startsWith("/content-builder")) return "content-builder";
  if (path.startsWith("/registry")) return "registry";
  if (path.startsWith("/notifications")) return "notifications";
  if (path.startsWith("/admin")) return "admin";
  if (path.startsWith("/approvals")) return "approvals";
  if (path.startsWith("/audit")) return "audit";
  if (path.startsWith("/operations")) return "operations";
  if (path === "/templates-v2" || path.startsWith("/templates-v2/")) return "templates-v2";
  // Legacy /templates path redirects to templates-v2
  if (path === "/templates" || path.startsWith("/templates/")) return "templates-v2";

  return "operations";
}

export function pathFromView(view: WorkspaceView): string {
  switch (view) {
    case "library":
      return documentsBaseByView["library"];
    case "my-docs":
      return documentsBaseByView["my-docs"];
    case "recent":
      return documentsBaseByView["recent"];
    case "create":
      // Legacy wizard `/create` depends on v1 profile/area endpoints that
      // are not wired. Redirect sidebar + primary-button "Novo documento"
      // clicks to the v2 create flow instead.
      return "/documents-v2/new";
    case "content-builder":
      return "/content-builder";
    case "registry":
      return "/registry";
    case "notifications":
      return "/notifications";
    case "admin":
      return "/admin";
    case "approvals":
      return "/approvals";
    case "audit":
      return "/audit";
    case "operations":
      return "/";
    case "templates-v2":
      return "/templates-v2";
    case "documents-v2":
      return "/documents-v2/new";
    default:
      return "/";
  }
}

export function isPathForView(pathname: string, view: WorkspaceView): boolean {
  const path = normalizePath(pathname);

  if (view === "library") {
    return path === "/documents" || (path.startsWith("/documents/") && !path.startsWith("/documents/mine") && !path.startsWith("/documents/recent"));
  }

  if (view === "my-docs") return path.startsWith("/documents/mine");
  if (view === "recent") return path.startsWith("/documents/recent");
  if (view === "create") return path.startsWith("/create");
  if (view === "content-builder") return path.startsWith("/content-builder");
  if (view === "registry") return path.startsWith("/registry");
  if (view === "notifications") return path.startsWith("/notifications");
  if (view === "admin") return path.startsWith("/admin");
  if (view === "approvals") return path.startsWith("/approvals");
  if (view === "audit") return path.startsWith("/audit");
  if (view === "operations") return path === "/" || path.startsWith("/operations");
  if (view === "templates-v2") return path === "/templates-v2" || path.startsWith("/templates-v2/");
  if (view === "documents-v2") return path === "/documents-v2" || path === "/documents-v2/new" || path.startsWith("/documents-v2/");

  return false;
}

export function parseDocumentsRoute(scopeView: DocumentsScopeView, pathname: string): DocumentsRoute {
  const basePath = documentsBasePath(scopeView);
  const path = normalizePath(pathname);

  if (!path.startsWith(basePath)) {
    return { view: "overview" };
  }

  const rest = path.slice(basePath.length).replace(/^\/+/, "");
  if (!rest) {
    return { view: "overview" };
  }

  if (rest === "all") {
    return { view: "collection" };
  }

  if (rest.startsWith("area/")) {
    return { view: "collection", areaCode: decodeURIComponent(rest.slice("area/".length)) };
  }

  if (rest.startsWith("type/")) {
    return { view: "collection", profileCode: decodeURIComponent(rest.slice("type/".length)) };
  }

  if (rest.startsWith("doc/")) {
    return { view: "detail", documentId: decodeURIComponent(rest.slice("doc/".length)) };
  }

  return { view: "overview" };
}

export function buildDocumentsPath(
  scopeView: DocumentsScopeView,
  target: { view: "overview" } | { view: "collection"; areaCode?: string; profileCode?: string } | { view: "detail"; documentId: string },
): string {
  const basePath = documentsBasePath(scopeView);

  if (target.view === "overview") {
    return basePath;
  }

  if (target.view === "detail") {
    return `${basePath}/doc/${encodeURIComponent(target.documentId)}`;
  }

  if (target.areaCode) {
    return `${basePath}/area/${encodeURIComponent(target.areaCode)}`;
  }

  if (target.profileCode) {
    return `${basePath}/type/${encodeURIComponent(target.profileCode)}`;
  }

  return `${basePath}/all`;
}

// ---------------------------------------------------------------------------
// Template editor route helpers
// ---------------------------------------------------------------------------

export type TemplateEditorParams = {
  profileCode: string;
  templateKey: string;
};

/**
 * Returns true when the current pathname targets the template editor.
 * Pattern: /registry/profiles/:profileCode/templates/:templateKey/edit
 */
export function isTemplateEditorPath(pathname: string): boolean {
  return parseTemplateEditorPath(pathname) !== null;
}

/**
 * Parses `/registry/profiles/:profileCode/templates/:templateKey/edit`
 * and returns the params, or null if the path does not match.
 */
export function parseTemplateEditorPath(pathname: string): TemplateEditorParams | null {
  const path = normalizePath(pathname);
  const match = /^\/registry\/profiles\/([^/]+)\/templates\/([^/]+)\/edit$/.exec(path);
  if (!match) return null;
  return {
    profileCode: decodeURIComponent(match[1]),
    templateKey: decodeURIComponent(match[2]),
  };
}

/**
 * Builds the template editor path from params.
 */
export function buildTemplateEditorPath(params: TemplateEditorParams): string {
  return `/registry/profiles/${encodeURIComponent(params.profileCode)}/templates/${encodeURIComponent(params.templateKey)}/edit`;
}

export type { DocumentsRoute, DocumentsScopeView };

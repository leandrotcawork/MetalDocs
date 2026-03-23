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

export function viewFromPath(pathname: string): WorkspaceView {
  const path = normalizePath(pathname);

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
      return "/create";
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

  return false;
}

export type { DocumentsScopeView };

import { useEffect, useMemo, useRef, useState } from "react";
import { buildProfileAccordions } from "../features/documents/adapters/catalogSummary";
import type { DocumentProfileItem, ProcessAreaItem, SearchDocumentItem } from "../lib.types";

export type WorkspaceView = "operations" | "approvals" | "audit" | "library" | "my-docs" | "recent" | "create" | "content-builder" | "registry" | "notifications" | "admin";

type WorkspaceShellProps = {
  userDisplayName: string;
  userRoleLabel: string;
  organizationLabel: string;
  activeView: WorkspaceView;
  searchValue: string;
  notificationsPending: number;
  documentCount: number;
  reviewCount: number;
  registryCount: number;
  showAdmin: boolean;
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  documents: SearchDocumentItem[];
  onSearchChange: (value: string) => void;
  onNavigate: (view: WorkspaceView) => void;
  onPrimaryAction: () => void;
  onRefreshWorkspace: () => void | Promise<void>;
  isRefreshing: boolean;
  onLogout: () => void | Promise<void>;
  children: React.ReactNode;
};

type NavSubItem = {
  key: WorkspaceView;
  label: string;
  icon: React.ReactNode;
  badge?: string;
  accent?: "default" | "danger" | "warning";
};

type NavSection = {
  label: string;
  items: NavSubItem[];
};

function sections(props: WorkspaceShellProps): NavSection[] {
  const overview: NavSection = {
    label: "Visao geral",
    items: [
      {
        key: "operations",
        label: "Dashboard",
        icon: (
          <svg viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.4">
            <rect x="1.5" y="1.5" width="5" height="5" rx="1.5" />
            <rect x="8.5" y="1.5" width="5" height="5" rx="1.5" />
            <rect x="1.5" y="8.5" width="5" height="5" rx="1.5" />
            <rect x="8.5" y="8.5" width="5" height="5" rx="1.5" />
          </svg>
        ),
      },
      {
        key: "approvals",
        label: "Aprovacoes",
        badge: String(props.reviewCount),
        accent: props.reviewCount > 0 ? "danger" : "default",
        icon: (
          <svg viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.4">
            <circle cx="7.5" cy="7.5" r="5.5" />
            <path d="M7.5 5v3l2 1.5" strokeLinecap="round" />
          </svg>
        ),
      },
      {
        key: "audit",
        label: "Audit trail",
        icon: (
          <svg viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.4">
            <path d="M1.5 3.5h12M1.5 7.5h8M1.5 11.5h5" strokeLinecap="round" />
          </svg>
        ),
      },
    ],
  };

  const documents: NavSection = {
    label: "Documentos",
    items: [
      {
        key: "library",
        label: "Todos Documentos",
        badge: String(props.documentCount),
        icon: (
          <svg viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.4">
            <path d="M2 2h11v11H2z" strokeLinejoin="round" />
            <path d="M5 5.5h5M5 8h5M5 10.5h3" strokeLinecap="round" />
          </svg>
        ),
      },
      {
        key: "my-docs",
        label: "Meus documentos",
        icon: (
          <svg viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.4">
            <circle cx="7.5" cy="5" r="2.5" />
            <path d="M2.5 13c0-2.8 2.2-5 5-5s5 2.2 5 5" strokeLinecap="round" />
          </svg>
        ),
      },
      {
        key: "recent",
        label: "Recentes",
        icon: (
          <svg viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.4">
            <circle cx="7.5" cy="7.5" r="5.5" />
            <path d="M7.5 4.5v3.5l2.5 1.5" strokeLinecap="round" />
          </svg>
        ),
      },
    ],
  };

  const tail: NavSection[] = [
    {
      label: "Workspace",
      items: [
        {
          key: "create",
          label: "Novo documento",
          icon: (
            <svg viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.4">
              <path d="M7.5 2v11M2 7.5h11" strokeLinecap="round" />
            </svg>
          ),
        },
        {
          key: "registry",
          label: "Tipos documentais",
          badge: String(props.registryCount),
          icon: (
            <svg viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.4">
              <circle cx="4.5" cy="4.5" r="1.5" />
              <circle cx="10.5" cy="4.5" r="1.5" />
              <circle cx="7.5" cy="10.5" r="1.5" />
              <path d="M5.8 5.2 6.9 9M9.2 5.2 8.1 9" strokeLinecap="round" />
            </svg>
          ),
        },
      ],
    },
  ];

  if (props.showAdmin) {
    tail.push({
      label: "Admin",
      items: [
        {
          key: "admin",
          label: "Usuarios internos",
          icon: (
            <svg viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.4">
              <circle cx="7.5" cy="5" r="2.5" />
              <path d="M2.5 13c0-2.8 2.2-5 5-5s5 2.2 5 5" strokeLinecap="round" />
            </svg>
          ),
        },
      ],
    });
  }

  return [overview, documents, ...tail];
}

function activeTitle(activeView: WorkspaceView): string {
  switch (activeView) {
    case "operations":
      return "Centro Operacional";
    case "approvals":
      return "Aprovacoes";
    case "audit":
      return "Audit Trail";
    case "library":
      return "Todos Documentos";
    case "my-docs":
      return "Meus Documentos";
    case "recent":
      return "Recentes";
    case "create":
      return "Novo Documento";
    case "content-builder":
      return "Editor de Conteudo";
    case "registry":
      return "Tipos Documentais";
    case "notifications":
      return "Notificacoes";
    case "admin":
      return "Usuarios Internos";
    default:
      return "Workspace";
  }
}

function isDocumentCatalogView(activeView: WorkspaceView): boolean {
  return activeView === "library" || activeView === "my-docs" || activeView === "recent";
}

export function DocumentWorkspaceShell(props: WorkspaceShellProps) {
  const navSections = sections(props);
  const primarySections = navSections.slice(0, 2);
  const secondarySections = navSections.slice(2);
  const typedSections = useMemo(
    () => buildProfileAccordions(props.documentProfiles, props.documents, props.processAreas),
    [props.documentProfiles, props.documents, props.processAreas],
  );
  const currentTitle = activeTitle(props.activeView);
  const isCreateView = props.activeView === "create";
  const isContentBuilder = props.activeView === "content-builder";
  const isCatalogView = isDocumentCatalogView(props.activeView);
  const userMenuRef = useRef<HTMLDivElement | null>(null);
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({
    po: true,
    it: false,
    rg: false,
    fm: false,
  });
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  useEffect(() => {
    function handlePointerDown(event: MouseEvent) {
      if (!userMenuRef.current) {
        return;
      }
      if (!userMenuRef.current.contains(event.target as Node)) {
        setUserMenuOpen(false);
      }
    }

    function handleKeydown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setUserMenuOpen(false);
      }
    }

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeydown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeydown);
    };
  }, []);

  function toggleSection(sectionKey: string) {
    setOpenSections((current) => ({
      ...current,
      [sectionKey]: !current[sectionKey],
    }));
  }

  return (
    <div className="workspace-shell">
      <header className="workspace-topbar">
        <div className="workspace-brand">
          <div className="workspace-brand-mark">
            <svg width="15" height="15" viewBox="0 0 15 15" fill="none" stroke="rgba(255,255,255,0.9)" strokeWidth="1.3">
              <path d="M3 2h6.5L12 4.5V13H3V2z" strokeLinejoin="round" />
              <path d="M9.5 2v2.5H12" strokeLinejoin="round" />
              <path d="M5 7h5M5 9.5h5M5 12h3" strokeLinecap="round" />
            </svg>
          </div>
          <div className="workspace-brand-text">
            <div className="workspace-brand-name">MetalDocs</div>
            <div className="workspace-brand-sub">SGQ / {props.organizationLabel}</div>
          </div>
        </div>

        <div className="workspace-search">
          <span className="workspace-search-icon" aria-hidden="true">
            <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.5">
              <circle cx="5.5" cy="5.5" r="4" />
              <path d="M9 9l2.5 2.5" strokeLinecap="round" />
            </svg>
          </span>
          <input
            value={props.searchValue}
            onChange={(event) => props.onSearchChange(event.target.value)}
            placeholder="Buscar por titulo, codigo, area..."
          />
          <span className="workspace-search-kbd">CTRL+K</span>
        </div>

        <div className="workspace-topbar-actions">
          <button
            type="button"
            className="workspace-icon-button"
            title="Atualizar workspace"
            onClick={() => void props.onRefreshWorkspace()}
            disabled={props.isRefreshing}
          >
            <span aria-hidden="true">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <path d="M11.5 3.5v3h-3" strokeLinecap="round" strokeLinejoin="round" />
                <path d="M2.5 10.5v-3h3" strokeLinecap="round" strokeLinejoin="round" />
                <path d="M11 6A4.5 4.5 0 0 0 3.7 3.1L2.5 4.3M3 8A4.5 4.5 0 0 0 10.3 10.9l1.2-1.2" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </span>
          </button>
          <button type="button" className="workspace-icon-button" title="Notificacoes" onClick={() => props.onNavigate("notifications")}>
            <span aria-hidden="true">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <path d="M7 1.5a4 4 0 0 1 4 4v2.5l1 1.5H2L3 8V5.5a4 4 0 0 1 4-4z" />
                <path d="M5.5 11.5a1.5 1.5 0 0 0 3 0" strokeLinecap="round" />
              </svg>
            </span>
            {props.notificationsPending > 0 && <span className="workspace-dot" />}
          </button>
          <button type="button" className="workspace-primary-button" onClick={props.onPrimaryAction}>
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M6 1v10M1 6h10" strokeLinecap="round" />
            </svg>
            Novo documento
          </button>
          <div className="workspace-user-menu" ref={userMenuRef} data-open={userMenuOpen ? "true" : "false"}>
            <button
              type="button"
              className="workspace-user-trigger"
              aria-haspopup="menu"
              aria-expanded={userMenuOpen}
              onClick={() => setUserMenuOpen((current) => !current)}
              title={props.userDisplayName}
            >
              <div className="workspace-avatar">{props.userDisplayName.slice(0, 2).toUpperCase()}</div>
              <span className="workspace-user-copy">
                <strong>{props.userDisplayName}</strong>
                <small>{props.userRoleLabel}</small>
              </span>
              <span className="workspace-user-chevron" aria-hidden="true">
                <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.4">
                  <path d="M3 4.5 6 7.5l3-3" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
              </span>
            </button>

            {userMenuOpen && (
              <div className="workspace-user-dropdown" role="menu">
                <div className="workspace-user-meta">
                  <span>Workspace</span>
                  <strong>{props.organizationLabel}</strong>
                </div>
                <button
                  data-testid="logout-button"
                  type="button"
                  className="workspace-user-item is-danger"
                  role="menuitem"
                  onClick={() => {
                    setUserMenuOpen(false);
                    void props.onLogout();
                  }}
                >
                  Sair da sessao
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      <div className="workspace-layout">
        <aside className="workspace-sidebar">
          <div className="workspace-sidebar-scroll">
            <div className="workspace-context-pill">
              <div className="workspace-context-mark" aria-hidden="true">
                <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.4">
                  <path d="M2 2h9v9H2z" strokeLinejoin="round" />
                  <path d="M2 5h9M5 5v6" strokeLinecap="round" />
                </svg>
              </div>
              <div className="workspace-context-copy">
                <strong>{props.organizationLabel}</strong>
                <small>{props.userRoleLabel}</small>
              </div>
              <span className="workspace-context-chevron" aria-hidden="true">
                <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.4">
                  <path d="M3 4.5 6 7.5l3-3" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
              </span>
            </div>

            <div className="workspace-divider" />

            {primarySections.map((section, index) => (
              <div key={section.label} className="workspace-nav-group">
                <div className="workspace-sidebar-section-label">{section.label}</div>
                <div className="workspace-flat-nav">
                  {section.items.map((item) => (
                    <button
                      key={`${section.label}-${item.label}`}
                      type="button"
                      className={`workspace-flat-nav-item ${props.activeView === item.key ? "is-active" : ""}`}
                      onClick={() => props.onNavigate(item.key)}
                    >
                      <span className="workspace-flat-nav-main">
                        <span className="workspace-nav-icon">{item.icon}</span>
                        <span className="workspace-flat-nav-label">{item.label}</span>
                      </span>
                      {item.badge && <span className={`workspace-nav-badge ${item.accent === "danger" ? "is-danger" : item.accent === "warning" ? "is-warning" : ""}`}>{item.badge}</span>}
                    </button>
                  ))}
                </div>
                {(index === 0 || index === 1) && <div className="workspace-divider" />}
              </div>
            ))}

            <div className="workspace-sidebar-section-label">Por tipo</div>
            {typedSections.map((section) => {
              const isOpen = openSections[section.code] ?? false;
              return (
                <div key={section.code} className="workspace-accordion">
                  <button type="button" className={`workspace-section-trigger ${isOpen ? "is-open" : ""}`} onClick={() => toggleSection(section.code)}>
                    <span className="workspace-section-chevron" aria-hidden="true">
                      <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.4">
                        <path d="M4.5 3 7.5 6l-3 3" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                    </span>
                    <span className="workspace-section-label" title={props.documentProfiles.find((item) => item.code === section.code)?.name ?? section.label}>{section.label}</span>
                    <span className="workspace-section-count">{section.count}</span>
                  </button>

                  {isOpen && (
                    <div className="workspace-subnav typed">
                      <button type="button" className={`workspace-subnav-item ${props.activeView === "library" ? "is-active" : ""}`} onClick={() => props.onNavigate("library")}>
                        <span className="workspace-subnav-main">
                          <span className="workspace-subnav-label">Todos</span>
                        </span>
                        <span className="workspace-sub-count">{section.count}</span>
                      </button>
                      {section.areas.map((area) => (
                        <button key={`${section.code}-${area.label}`} type="button" className="workspace-subnav-item" onClick={() => props.onNavigate("library")}>
                          <span className="workspace-subnav-main">
                            <span className={`workspace-sub-dot profile-${section.code}`} />
                            <span className="workspace-subnav-label">{area.label}</span>
                          </span>
                          <span className="workspace-sub-count">{area.count}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}

            {typedSections.length > 0 && <div className="workspace-divider" />}

            {secondarySections.map((section) => (
              <div key={section.label} className="workspace-nav-group">
                <div className="workspace-sidebar-section-label">{section.label}</div>
                <div className="workspace-flat-nav">
                  {section.items.map((item) => (
                    <button
                      key={`${section.label}-${item.label}`}
                      type="button"
                      className={`workspace-flat-nav-item ${props.activeView === item.key ? "is-active" : ""}`}
                      onClick={() => props.onNavigate(item.key)}
                    >
                      <span className="workspace-flat-nav-main">
                        <span className="workspace-nav-icon">{item.icon}</span>
                        <span className="workspace-flat-nav-label">{item.label}</span>
                      </span>
                      {item.badge && <span className={`workspace-nav-badge ${item.accent === "danger" ? "is-danger" : item.accent === "warning" ? "is-warning" : ""}`}>{item.badge}</span>}
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>

          <div className="workspace-sidebar-footer">
            <div className="workspace-runtime-card">
              <span>Workspace</span>
              <strong>Control room documental</strong>
              <small>{props.reviewCount} documento(s) em revisao</small>
            </div>
          </div>
        </aside>

        <main className={`workspace-main ${isCatalogView || isCreateView || isContentBuilder ? "is-toolbarless" : ""} ${isCreateView ? "is-create-view" : ""}`}>
          {!isCatalogView && !isCreateView && !isContentBuilder && (
            <div className="workspace-toolbar workspace-toolbar-as-panel">
              <div className="workspace-toolbar-top">
                <p className="workspace-toolbar-kicker">Workspace</p>
                <div className="workspace-breadcrumb">
                  <strong>{currentTitle}</strong>
                </div>
              </div>
            </div>
          )}
          {isCreateView || isContentBuilder
            ? props.children
            : <div className={`workspace-content ${isCatalogView ? "is-toolbarless" : ""}`}>{props.children}</div>}
        </main>
      </div>
    </div>
  );
}

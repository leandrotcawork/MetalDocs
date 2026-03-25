import { useCallback, useEffect, useMemo } from "react";
import { buildDocumentProfileCountMap } from "./adapters/catalogSummary";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { metalNobreProcessAreaHint } from "./adapters/metalNobreExperience";
import { formatDocumentDisplayName } from "../shared/documentDisplay";
import { type RecentDocumentItem, useDocumentsStore } from "../../store/documents.store";
import { FilterDropdown, type SelectMenuOption } from "../../components/ui/FilterDropdown";
import { DocumentsHubHeader } from "./DocumentsHubHeader";
import { buildDocumentsPath, documentsBasePath, parseDocumentsRoute } from "../../routing/workspaceRoutes";
import styles from "./DocumentsHubView.module.css";
import type { DocumentListItem, DocumentProfileGovernanceItem, DocumentProfileItem, ManagedUserItem, ProcessAreaItem, SearchDocumentItem } from "../../lib.types";

type DocumentsHubViewProps = {
  view: "library" | "my-docs" | "recent";
  loadState: "idle" | "loading" | "ready" | "error";
  currentUserId?: string;
  managedUsers: ManagedUserItem[];
  documents: SearchDocumentItem[];
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  selectedDocument: DocumentListItem | null;
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  searchQuery: string;
  formatDate: (value?: string) => string;
  onSearchQueryChange: (value: string) => void;
  onCreateDocument: () => void;
  onOpenDocument: (documentId: string, nextView?: "library" | "content-builder") => void | Promise<void>;
  onOpenDocumentForHub: (documentId: string) => void | Promise<void>;
};

type HubScope = "all" | "mine" | "recent";
type HubStatus = "all" | "draft" | "review" | "approved";
type HubMode = "card" | "list";
type CollectionSort = "created-desc" | "created-asc" | "code-asc" | "code-desc";
const recentKeyPrefix = "metaldocs.recentDocuments";

function recentStorageKey(userId?: string) {
  if (!userId) return null;
  return `${recentKeyPrefix}.${userId}`;
}

function hydrateRecentDocuments(items: RecentDocumentItem[], documents: SearchDocumentItem[]): RecentDocumentItem[] {
  if (items.length === 0 || documents.length === 0) return items;
  const byId = new Map(documents.map((doc) => [doc.documentId, doc] as const));
  return items.map((item) => {
    const match = byId.get(item.documentId);
    if (!match) return item;
    return {
      ...match,
      openedAt: item.openedAt || item.createdAt || match.createdAt || new Date().toISOString(),
    };
  });
}

function loadRecentDocuments(userId?: string): RecentDocumentItem[] {
  const key = recentStorageKey(userId);
  if (!key) return [];
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item) => item && typeof item.documentId === "string");
  } catch {
    return [];
  }
}

function storeRecentDocuments(userId: string | undefined, items: RecentDocumentItem[]) {
  const key = recentStorageKey(userId);
  if (!key) return;
  window.localStorage.setItem(key, JSON.stringify(items));
}

function documentScope(view: DocumentsHubViewProps["view"]): HubScope {
  if (view === "my-docs") return "mine";
  if (view === "recent") return "recent";
  return "all";
}

function normalizeAreaCode(value?: string): string {
  return (value ?? "sem-area").trim().toLowerCase();
}

function profileBadgeText(profile: DocumentProfileItem): string {
  const alias = profile.alias?.trim?.();
  const code = profile.code?.trim?.();
  const name = profile.name?.trim?.();

  const shortAlias = alias && alias.length <= 3 ? alias.toUpperCase() : "";
  if (shortAlias) return shortAlias;

  const shortCode = code && code.length <= 3 ? code.toUpperCase() : "";
  if (shortCode) return shortCode;

  const source = (alias || code || name || "").toUpperCase();
  const words = source.split(/[^A-Z0-9]+/).filter(Boolean);
  if (words.length >= 2) return `${words[0][0]}${words[1][0]}`;
  if (words.length === 1 && words[0].length >= 2) return words[0].slice(0, 2);
  return source.slice(0, 2) || "--";
}

function statusLabel(status: string): string {
  switch (status) {
    case "IN_REVIEW":
      return "Em revisao";
    case "APPROVED":
    case "PUBLISHED":
      return "Aprovado";
    case "ARCHIVED":
      return "Arquivado";
    default:
      return "Draft";
  }
}

function normalizeHubStatus(value: string | null): HubStatus {
  if (value === "draft" || value === "review" || value === "approved") return value;
  return "all";
}

function normalizeHubMode(value: string | null): HubMode {
  if (value === "list") return "list";
  return "card";
}

function normalizeCollectionSort(value: string | null): CollectionSort {
  if (value === "created-asc" || value === "code-asc" || value === "code-desc") return value;
  return "created-desc";
}

function formatDateOnly(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleDateString("pt-BR");
}

function normalizeToken(value?: string): string {
  return (value ?? "").trim().toLowerCase();
}

const departmentColorByCode: Record<string, string> = {
  operacoes: "#2D5DAD",
  qualidade: "#1F7A45",
  quality: "#1F7A45",
  comercial: "#9D2335",
  commercial: "#9D2335",
  financeiro: "#8A5800",
  finance: "#8A5800",
  logistica: "#2B5C8A",
  logistics: "#2B5C8A",
  compras: "#6B3A9C",
  purchasing: "#6B3A9C",
};

const areaColorByCode: Record<string, string> = {
  quality: "#1F7A45",
  marketplaces: "#6B3A9C",
  commercial: "#9D2335",
  purchasing: "#8A5800",
  logistics: "#2B5C8A",
  finance: "#B4541A",
  "sem-area": "#6A5C62",
};

function hexToRgba(hex: string, alpha: number): string {
  const value = hex.replace("#", "");
  if (!/^[0-9a-fA-F]{6}$/.test(value)) return `rgba(106, 92, 98, ${alpha})`;
  const red = Number.parseInt(value.slice(0, 2), 16);
  const green = Number.parseInt(value.slice(2, 4), 16);
  const blue = Number.parseInt(value.slice(4, 6), 16);
  return `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}

function compareDocumentCode(
  left: SearchDocumentItem,
  right: SearchDocumentItem,
  direction: "asc" | "desc",
  profiles: DocumentProfileItem[],
): number {
  const leftSequence = left.documentSequence ?? Number.MAX_SAFE_INTEGER;
  const rightSequence = right.documentSequence ?? Number.MAX_SAFE_INTEGER;

  if (leftSequence !== rightSequence) {
    return direction === "asc" ? leftSequence - rightSequence : rightSequence - leftSequence;
  }

  const leftName = formatDocumentDisplayName(left, profiles).toLowerCase();
  const rightName = formatDocumentDisplayName(right, profiles).toLowerCase();
  return direction === "asc" ? leftName.localeCompare(rightName) : rightName.localeCompare(leftName);
}

export function DocumentsHubView(props: DocumentsHubViewProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const scope = documentScope(props.view);
  const {
    documentsHubView,
    documentsHubMode,
    documentsHubStatus,
    documentsHubArea,
    documentsHubProfile,
    setDocumentsHubView,
    setDocumentsHubMode,
    setDocumentsHubStatus,
    setDocumentsHubArea,
    setDocumentsHubProfile,
    recentDocuments,
    setRecentDocuments,
  } = useDocumentsStore();

  const basePath = documentsBasePath(props.view);
  const route = useMemo(() => parseDocumentsRoute(props.view, location.pathname), [location.pathname, props.view]);
  const queryStatus = useMemo(() => normalizeHubStatus(searchParams.get("status")), [searchParams]);
  const queryMode = useMemo(() => normalizeHubMode(searchParams.get("mode")), [searchParams]);
  const queryDepartment = useMemo(() => searchParams.get("department") ?? "all", [searchParams]);
  const queryAreaFilter = useMemo(() => searchParams.get("areaFilter") ?? "all", [searchParams]);
  const querySort = useMemo(() => normalizeCollectionSort(searchParams.get("sort")), [searchParams]);
  const queryText = useMemo(() => searchParams.get("q") ?? "", [searchParams]);

  const buildHubParams = useCallback(
    (next?: Partial<{ status: HubStatus; mode: HubMode; q: string; department: string; areaFilter: string; sort: CollectionSort }>) => {
      const status = next?.status ?? documentsHubStatus;
      const mode = next?.mode ?? documentsHubMode;
      const q = next?.q ?? props.searchQuery;
      const department = next?.department ?? queryDepartment;
      const areaFilter = next?.areaFilter ?? queryAreaFilter;
      const sort = next?.sort ?? querySort;
      const params = new URLSearchParams();
      if (status !== "all") params.set("status", status);
      if (mode !== "card") params.set("mode", mode);
      if (department !== "all") params.set("department", department);
      if (areaFilter !== "all") params.set("areaFilter", areaFilter);
      if (sort !== "created-desc") params.set("sort", sort);
      if (q.trim()) params.set("q", q.trim());
      return params;
    },
    [documentsHubMode, documentsHubStatus, props.searchQuery, queryAreaFilter, queryDepartment, querySort],
  );

  const navigateWithParams = useCallback(
    (path: string, options?: { replace?: boolean }) => {
      const params = buildHubParams();
      const suffix = params.toString();
      navigate(suffix ? `${path}?${suffix}` : path, options);
    },
    [buildHubParams, navigate],
  );

  const handleSearchQueryChange = useCallback(
    (value: string) => {
      props.onSearchQueryChange(value);
      const nextParams = buildHubParams({ q: value });
      setSearchParams(nextParams, { replace: true });
    },
    [buildHubParams, props, setSearchParams],
  );

  const headerTitle = scope === "mine" ? "Meus documentos" : scope === "recent" ? "Recentes" : "Todos documentos";
  const headerVariant = documentsHubView === "collection" || documentsHubView === "detail" ? "compact" : "default";
  const headerShell = (
    <DocumentsHubHeader
      title={headerTitle}
      searchQuery={props.searchQuery}
      onSearchQueryChange={handleSearchQueryChange}
      variant={headerVariant}
    />
  );

  const scopedDocuments = useMemo(() => {
    if (scope === "mine") {
      return props.documents.filter((item) => item.ownerId === props.currentUserId);
    }
    if (scope === "recent") {
      if (recentDocuments.length > 0) {
        return recentDocuments;
      }
      return [...props.documents].sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime());
    }
    return props.documents;
  }, [props.currentUserId, props.documents, recentDocuments, scope]);

  useEffect(() => {
    if (!props.currentUserId) return;
    setRecentDocuments(loadRecentDocuments(props.currentUserId));
  }, [props.currentUserId, setRecentDocuments]);

  useEffect(() => {
    if (!props.currentUserId) return;
    if (recentDocuments.length === 0) return;
    if (props.documents.length === 0) return;

    const hydrated = hydrateRecentDocuments(recentDocuments, props.documents);
    const needsUpgrade = hydrated.some((item, index) => {
      const previous = recentDocuments[index];
      if (!previous) return true;
      return (
        item.documentCode !== previous.documentCode ||
        item.documentSequence !== previous.documentSequence ||
        item.title !== previous.title ||
        item.openedAt !== previous.openedAt
      );
    });

    if (!needsUpgrade) return;
    setRecentDocuments(hydrated);
    storeRecentDocuments(props.currentUserId, hydrated);
  }, [props.currentUserId, props.documents, recentDocuments, setRecentDocuments]);

  useEffect(() => {
    if (route.view === "overview") {
      setDocumentsHubView("overview");
      setDocumentsHubStatus("all");
      setDocumentsHubArea("all");
      setDocumentsHubProfile("all");
      return;
    }

    if (route.view === "collection") {
      setDocumentsHubView("collection");
      if (route.areaCode) {
        setDocumentsHubArea(route.areaCode);
        setDocumentsHubProfile("all");
      } else if (route.profileCode) {
        setDocumentsHubProfile(route.profileCode);
        setDocumentsHubArea("all");
      } else {
        setDocumentsHubArea("all");
        setDocumentsHubProfile("all");
      }
      return;
    }

    if (route.view === "detail") {
      setDocumentsHubView("detail");
      if (route.documentId && props.selectedDocument?.documentId !== route.documentId) {
        void props.onOpenDocumentForHub(route.documentId);
      }
    }
  }, [props.selectedDocument?.documentId, props.view, route, setDocumentsHubArea, setDocumentsHubProfile, setDocumentsHubStatus, setDocumentsHubView]);

  useEffect(() => {
    if (queryStatus !== documentsHubStatus) {
      setDocumentsHubStatus(queryStatus);
    }
    if (queryMode !== documentsHubMode) {
      setDocumentsHubMode(queryMode);
    }
    if (queryText !== props.searchQuery) {
      props.onSearchQueryChange(queryText);
    }
  }, [documentsHubMode, documentsHubStatus, props, queryMode, queryStatus, queryText, setDocumentsHubMode, setDocumentsHubStatus]);

  const recentFallback = useMemo(
    () => scopedDocuments.slice(0, 8).map((item) => ({ ...item, openedAt: item.createdAt })),
    [scopedDocuments],
  );
  const recentItems = useMemo(() => {
    const base = recentDocuments.length > 0 ? recentDocuments : recentFallback;
    return hydrateRecentDocuments(base, props.documents);
  }, [props.documents, recentDocuments, recentFallback]);
  const profileCounts = useMemo(() => buildDocumentProfileCountMap(scopedDocuments), [scopedDocuments]);
  const processAreaNameByCode = useMemo(
    () => new Map(props.processAreas.map((item) => [normalizeAreaCode(item.code), item.name] as const)),
    [props.processAreas],
  );
  const profileNameByCode = useMemo(
    () => new Map(props.documentProfiles.map((item) => [item.code, item.name] as const)),
    [props.documentProfiles],
  );
  const userNameById = useMemo(
    () => new Map(props.managedUsers.map((item) => [item.userId, item.displayName] as const)),
    [props.managedUsers],
  );
  const departmentOptions = useMemo(
    () =>
      Array.from(
        new Set(scopedDocuments.map((item) => item.department.trim()).filter((value) => value.length > 0)),
      ).sort((left, right) => left.localeCompare(right)),
    [scopedDocuments],
  );
  const areaOptions = useMemo(
    () =>
      Array.from(new Set(scopedDocuments.map((item) => normalizeAreaCode(item.processArea)))).sort((left, right) => {
        const leftLabel = processAreaNameByCode.get(left) ?? left;
        const rightLabel = processAreaNameByCode.get(right) ?? right;
        return leftLabel.localeCompare(rightLabel);
      }),
    [processAreaNameByCode, scopedDocuments],
  );
  const departmentFilterOptions = useMemo<SelectMenuOption[]>(
    () => [
      { value: "all", label: "Departamento: todos" },
      ...departmentOptions.map((department) => ({ value: department, label: department })),
    ],
    [departmentOptions],
  );
  const areaFilterOptions = useMemo<SelectMenuOption[]>(
    () => [
      { value: "all", label: "Area: todas" },
      ...areaOptions.map((areaCode) => ({
        value: areaCode,
        label: processAreaNameByCode.get(areaCode) ?? areaCode,
      })),
    ],
    [areaOptions, processAreaNameByCode],
  );
  const sortOptions = useMemo<SelectMenuOption[]>(
    () => [
      { value: "created-desc", label: "Criacao mais recente" },
      { value: "created-asc", label: "Criacao mais antiga" },
      { value: "code-asc", label: "Codigo menor para maior" },
      { value: "code-desc", label: "Codigo maior para menor" },
    ],
    [],
  );

  const areaCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const document of scopedDocuments) {
      const key = normalizeAreaCode(document.processArea);
      counts[key] = (counts[key] ?? 0) + 1;
    }
    return counts;
  }, [scopedDocuments]);

  const totalDocuments = scopedDocuments.length;
  const areaColors = ["#9D2335", "#1F5A3F", "#6B3A9C", "#B4541A", "#2B5C8A", "#A32B6B"];
  const areaCards = useMemo(() => {
    const cards = props.processAreas.map((area, index) => ({
      code: normalizeAreaCode(area.code),
      label: area.name,
      count: areaCounts[normalizeAreaCode(area.code)] ?? 0,
      description: metalNobreProcessAreaHint(area.code),
      color: areaColors[index % areaColors.length],
    }));
    cards.unshift({
      code: "sem-area",
      label: "Sem area",
      count: areaCounts["sem-area"] ?? 0,
      description: "Sem classificacao atribuida.",
      color: areaColors[0],
    });
    return cards;
  }, [areaCounts, props.processAreas]);

  const profileCards = useMemo(() => {
    return props.documentProfiles
      .map((profile) => ({
        code: profile.code,
        label: profile.name,
        description: profile.description || "Perfil documental configurado no registry.",
        badge: profileBadgeText(profile),
        count: profileCounts[profile.code] ?? 0,
        color: areaColors[props.documentProfiles.findIndex((item) => item.code === profile.code) % areaColors.length],
      }))
      .filter((item) => item.count > 0);
  }, [profileCounts, props.documentProfiles]);

  const approvedCount = scopedDocuments.filter((item) => item.status === "APPROVED" || item.status === "PUBLISHED").length;
  const inReviewCount = scopedDocuments.filter((item) => item.status === "IN_REVIEW").length;
  const expiringSoon = scopedDocuments.filter((item) => {
    if (!item.expiryAt) return false;
    const expiry = new Date(item.expiryAt).getTime();
    const thirtyDays = 1000 * 60 * 60 * 24 * 30;
    return expiry - Date.now() <= thirtyDays && expiry > Date.now();
  }).length;
  const latestDocuments = useMemo(
    () =>
      [...scopedDocuments]
        .sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime())
        .slice(0, 5),
    [scopedDocuments],
  );
  const nextExpiryDays = useMemo(() => {
    const upcoming = scopedDocuments
      .map((item) => item.expiryAt)
      .filter((value): value is string => Boolean(value))
      .map((value) => Math.ceil((new Date(value).getTime() - Date.now()) / (1000 * 60 * 60 * 24)))
      .filter((value) => value >= 0);
    if (upcoming.length === 0) return 0;
    return Math.min(...upcoming);
  }, [scopedDocuments]);
  const focusAreaCard = useMemo(() => {
    return [...areaCards]
      .filter((item) => item.count > 0)
      .sort((left, right) => right.count - left.count)[0] ?? null;
  }, [areaCards]);


  const normalizedQuery = props.searchQuery.trim().toLowerCase();
  const baseFilteredDocuments = useMemo(() => {
    return scopedDocuments.filter((item) => {
      if (documentsHubArea !== "all") {
        const area = normalizeAreaCode(item.processArea);
        if (area !== documentsHubArea) return false;
      }
      if (documentsHubProfile !== "all" && item.documentProfile !== documentsHubProfile) return false;
      if (queryDepartment !== "all" && item.department !== queryDepartment) return false;
      if (queryAreaFilter !== "all" && normalizeAreaCode(item.processArea) !== queryAreaFilter) return false;
      if (!normalizedQuery) return true;
      const haystack = [
        item.title,
        item.documentId,
        item.documentCode,
        item.documentProfile,
        item.processArea,
        item.department,
        item.ownerId,
      ].join(" ").toLowerCase();
      return haystack.includes(normalizedQuery);
    });
  }, [documentsHubArea, documentsHubProfile, normalizedQuery, queryAreaFilter, queryDepartment, scopedDocuments]);

  const tabCounts = useMemo(() => ({
    all: baseFilteredDocuments.length,
    draft: baseFilteredDocuments.filter((item) => item.status === "DRAFT").length,
    review: baseFilteredDocuments.filter((item) => item.status === "IN_REVIEW").length,
    approved: baseFilteredDocuments.filter((item) => item.status === "APPROVED" || item.status === "PUBLISHED").length,
  }), [baseFilteredDocuments]);

  const collectionDocuments = useMemo(() => {
    let nextItems = baseFilteredDocuments;
    if (documentsHubStatus === "draft") {
      nextItems = nextItems.filter((item) => item.status === "DRAFT");
    }
    if (documentsHubStatus === "review") {
      nextItems = nextItems.filter((item) => item.status === "IN_REVIEW");
    }
    if (documentsHubStatus === "approved") {
      nextItems = nextItems.filter((item) => item.status === "APPROVED" || item.status === "PUBLISHED");
    }

    return [...nextItems].sort((left, right) => {
      if (querySort === "created-asc") {
        return new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime();
      }
      if (querySort === "code-asc") {
        return compareDocumentCode(left, right, "asc", props.documentProfiles);
      }
      if (querySort === "code-desc") {
        return compareDocumentCode(left, right, "desc", props.documentProfiles);
      }
      return new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime();
    });
  }, [baseFilteredDocuments, documentsHubStatus, props.documentProfiles, querySort]);

  const collectionTitle = useMemo(() => {
    if (documentsHubProfile !== "all") {
      return props.documentProfiles.find((item) => item.code === documentsHubProfile)?.name ?? documentsHubProfile;
    }
    if (documentsHubArea !== "all") {
      if (documentsHubArea === "sem-area") return "Sem area";
      return props.processAreas.find((item) => normalizeAreaCode(item.code) === documentsHubArea)?.name ?? documentsHubArea;
    }
    return headerTitle;
  }, [documentsHubArea, documentsHubProfile, headerTitle, props.documentProfiles, props.processAreas]);

  const handleRecentOpen = useCallback((item: SearchDocumentItem) => {
    const nextItems: RecentDocumentItem[] = [
      { ...item, openedAt: new Date().toISOString() },
      ...recentItems.filter((recent) => recent.documentId !== item.documentId),
    ].slice(0, 8);
    setRecentDocuments(nextItems);
    storeRecentDocuments(props.currentUserId, nextItems);
    navigateWithParams(buildDocumentsPath(props.view, { view: "detail", documentId: item.documentId }));
  }, [navigateWithParams, props.currentUserId, props.view, recentItems, setRecentDocuments]);

  const handleStatusChange = useCallback(
    (status: HubStatus) => {
      setDocumentsHubStatus(status);
      const nextParams = buildHubParams({ status });
      setSearchParams(nextParams, { replace: true });
    },
    [buildHubParams, setDocumentsHubStatus, setSearchParams],
  );

  const handleModeChange = useCallback(
    (mode: HubMode) => {
      setDocumentsHubMode(mode);
      const nextParams = buildHubParams({ mode });
      setSearchParams(nextParams, { replace: true });
    },
    [buildHubParams, setDocumentsHubMode, setSearchParams],
  );

  const handleDepartmentFilterChange = useCallback(
    (department: string) => {
      const nextParams = buildHubParams({ department });
      setSearchParams(nextParams, { replace: true });
    },
    [buildHubParams, setSearchParams],
  );

  const handleAreaFilterChange = useCallback(
    (areaFilter: string) => {
      const nextParams = buildHubParams({ areaFilter });
      setSearchParams(nextParams, { replace: true });
    },
    [buildHubParams, setSearchParams],
  );

  const handleSortChange = useCallback(
    (sort: CollectionSort) => {
      const nextParams = buildHubParams({ sort });
      setSearchParams(nextParams, { replace: true });
    },
    [buildHubParams, setSearchParams],
  );

  if (props.loadState === "loading") {
    return (
      <div className={styles.page}>
        {headerShell}
        <section className={styles.hub}>
          <div className={styles.state}>Carregando acervo...</div>
        </section>
      </div>
    );
  }

  if (props.loadState === "error") {
    return (
      <div className={styles.page}>
        {headerShell}
        <section className={styles.hub}>
          <div className={styles.state}>Falha ao carregar os documentos.</div>
        </section>
      </div>
    );
  }

  if (documentsHubView === "detail") {
    if (!props.selectedDocument) {
      return (
        <div className={styles.page}>
          {headerShell}
          <section className={styles.hub}>
            <div className={styles.state}>Selecione um documento para ver os detalhes.</div>
          </section>
        </div>
      );
    }

    const doc = props.selectedDocument;
    const profileLabel = props.documentProfiles.find((item) => item.code === doc.documentProfile)?.name ?? doc.documentProfile;
    const areaLabel = doc.processArea
      ? props.processAreas.find((item) => item.code === doc.processArea)?.name ?? doc.processArea
      : "Sem area";
    const governance = props.selectedProfileGovernance?.profileCode === doc.documentProfile
      ? props.selectedProfileGovernance
      : null;
    const ownerLabel = userNameById.get(doc.ownerId) ?? doc.ownerId ?? "-";
    const documentStatus = statusLabel(doc.status);

    return (
      <div className={styles.page}>
        {headerShell}
        <section className={styles.detail}>
          <div className={styles.detailShell}>
            <article className={styles.detailHeroCard}>
              <div className={styles.detailHeroLine} />
              <div className={styles.detailHeroMain}>
                <div className={styles.detailHeroTitleGroup}>
                  <h2>{formatDocumentDisplayName(doc, props.documentProfiles)}</h2>
                </div>
                <div className={styles.detailHeroAside}>
                  <span className={styles.detailDraftBadge}>
                    <span aria-hidden="true" className={styles.detailDraftDot} />
                    {documentStatus}
                  </span>
                </div>
              </div>
              <div className={styles.detailActionsBar}>
                <button type="button" className={styles.primaryButton} onClick={() => props.onOpenDocument(doc.documentId, "content-builder")}>
                  <svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                    <path d="M3.5 2.5h6l3 3v8H3.5z" />
                    <path d="M9.5 2.5v3h3" />
                    <path d="M6 9h4M6 11h3" strokeLinecap="round" />
                  </svg>
                  Abrir documento
                </button>
                <button type="button" className={styles.ghostButton} disabled>
                  <svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                    <path d="M2.5 8h3l2.5-3 2.5 6 3-4" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                  Enviar para revisao
                </button>
                <button type="button" className={styles.ghostButton} disabled>
                  <svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                    <path d="M6 2.5h5v9H6zM5 4H4v9h5" strokeLinejoin="round" />
                  </svg>
                  Duplicar
                </button>
                <button type="button" className={styles.ghostButton} disabled>
                  <svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                    <path d="M8 3a5 5 0 1 0 5 5" />
                    <path d="M8 1.8v2.4M8 8l1.7 1.2" strokeLinecap="round" />
                  </svg>
                  Historico de versoes
                </button>
              </div>
            </article>

            <div className={styles.detailPanelGrid}>
              <article className={`${styles.detailPanel} ${styles.detailPanelBlue}`}>
                <div className={styles.detailPanelHeader}>
                  <span className={styles.detailPanelIcon}>
                    <svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                      <rect x="2.5" y="2.5" width="11" height="11" rx="2.2" />
                      <path d="M5.2 8l1.8 1.8 3.7-3.7" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  </span>
                  <h3>Classificacao</h3>
                </div>
                <div className={styles.detailPanelBody}>
                  <div className={styles.detailPanelRow}><span>Familia</span><strong className={styles.detailChipBlue}>{doc.documentFamily || "-"}</strong></div>
                  <div className={styles.detailPanelRow}><span>Profile</span><strong>{profileLabel}</strong></div>
                  <div className={styles.detailPanelRow}><span>Departamento</span><strong>{doc.department || "-"}</strong></div>
                  <div className={styles.detailPanelRow}><span>Subject</span><strong>{doc.subject ?? "-"}</strong></div>
                </div>
              </article>

              <article className={`${styles.detailPanel} ${styles.detailPanelGreen}`}>
                <div className={styles.detailPanelHeader}>
                  <span className={styles.detailPanelIcon}>
                    <svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                      <path d="M8 2.2 12.8 4v3.6C12.8 10.3 10.7 12.8 8 13.8 5.3 12.8 3.2 10.3 3.2 7.6V4z" />
                    </svg>
                  </span>
                  <h3>Governanca</h3>
                </div>
                <div className={styles.detailPanelBody}>
                  <div className={styles.detailPanelRow}><span>Workflow</span><strong className={styles.detailChipGreen}>{governance?.workflowProfile ?? "-"}</strong></div>
                  <div className={styles.detailPanelRow}><span>Revisao</span><strong>{governance ? `${governance.reviewIntervalDays} dias` : "-"}</strong></div>
                  <div className={styles.detailPanelRow}><span>Aprovacao</span><strong className={styles.detailChipGold}>{governance ? (governance.approvalRequired ? "Obrigatoria" : "Opcional") : "-"}</strong></div>
                  <div className={styles.detailPanelRow}><span>Validade</span><strong>{governance ? `${governance.validityDays} dias` : "-"}</strong></div>
                </div>
              </article>

              <article className={`${styles.detailPanel} ${styles.detailPanelRose}`}>
                <div className={styles.detailPanelHeader}>
                  <span className={styles.detailPanelIcon}>
                    <svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                      <path d="M5.6 7.1a2 2 0 1 0 0-4 2 2 0 0 0 0 4ZM11 7.8a1.7 1.7 0 1 0 0-3.4 1.7 1.7 0 0 0 0 3.4Z" />
                      <path d="M2.8 12.8c.3-2 1.6-3 3.7-3s3.4 1 3.7 3M9.4 12.8c.2-1.3 1.1-2.1 2.5-2.1 1.4 0 2.3.8 2.5 2.1" strokeLinecap="round" />
                    </svg>
                  </span>
                  <h3>Colaboracao</h3>
                </div>
                <div className={styles.detailPanelBody}>
                  <div className={styles.detailPanelRow}><span>Lock de edicao</span><strong className={styles.detailMutedValue}>Sem lock ativo</strong></div>
                  <div className={styles.detailPanelRow}><span>Ativo desde</span><strong>{props.formatDate(new Date().toISOString())}</strong></div>
                  <div className={styles.detailPanelRow}><span>Proxima revisao</span><strong>{doc.expiryAt ? props.formatDate(doc.expiryAt) : "-"}</strong></div>
                  <div className={styles.detailPanelRow}><span>Autor</span><strong>{ownerLabel}</strong></div>
                </div>
              </article>

              <article className={`${styles.detailPanel} ${styles.detailPanelCyan}`}>
                <div className={styles.detailPanelHeader}>
                  <span className={styles.detailPanelIcon}>
                    <svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                      <path d="M4 2.8h8M4 6.2h8M4 9.6h5" strokeLinecap="round" />
                      <path d="m10.3 12.2 1.3 1.3 2.2-2.2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  </span>
                  <h3>Diff da versao atual</h3>
                </div>
                <div className={styles.detailDiffEmpty}>
                  <span className={styles.detailDiffIcon}>+</span>
                  <p>Nenhuma alteracao registrada nesta versao.</p>
                  <small>versao 1 · draft</small>
                </div>
              </article>
            </div>
          </div>
        </section>
      </div>
    );
  }

  if (documentsHubView === "collection") {
    return (
      <div className={styles.page}>
        {headerShell}
        <section className={styles.collection}>
          <div className={styles.collectionShell}>
            <div className={styles.collectionIntro}>
              <div className={styles.collectionHeader}>
                <h2>{collectionTitle} <span>({tabCounts.all})</span></h2>
                <div className={styles.collectionActions}>
                <div className={styles.viewToggle}>
                  <button
                    type="button"
                    className={documentsHubMode === "card" ? styles.isActive : ""}
                    onClick={() => handleModeChange("card")}
                    title="Exibir em cards"
                  >
                      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
                        <rect x="1" y="1" width="6" height="6" rx="1.5" />
                        <rect x="9" y="1" width="6" height="6" rx="1.5" />
                        <rect x="1" y="9" width="6" height="6" rx="1.5" />
                        <rect x="9" y="9" width="6" height="6" rx="1.5" />
                      </svg>
                    </button>
                  <button
                    type="button"
                    className={documentsHubMode === "list" ? styles.isActive : ""}
                    onClick={() => handleModeChange("list")}
                    title="Exibir em lista"
                  >
                      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
                        <path d="M3 4h10M3 8h10M3 12h10" strokeLinecap="round" />
                      </svg>
                    </button>
                  </div>
                  <button type="button" className={styles.collectionCreateButton} onClick={props.onCreateDocument}>
                    + Novo documento
                  </button>
                </div>
              </div>

            <div className={styles.collectionControlRow}>
              <div className={styles.tabRow}>
                <button type="button" className={documentsHubStatus === "all" ? styles.tabActive : styles.tab} onClick={() => handleStatusChange("all")}>Todos ({tabCounts.all})</button>
                <button type="button" className={documentsHubStatus === "draft" ? styles.tabActive : styles.tab} onClick={() => handleStatusChange("draft")}>Draft ({tabCounts.draft})</button>
                <button type="button" className={documentsHubStatus === "review" ? styles.tabActive : styles.tab} onClick={() => handleStatusChange("review")}>Em revisao ({tabCounts.review})</button>
                <button type="button" className={documentsHubStatus === "approved" ? styles.tabActive : styles.tab} onClick={() => handleStatusChange("approved")}>Aprovados ({tabCounts.approved})</button>
              </div>
              <div className={styles.collectionFilterBar}>
                <FilterDropdown
                  id="collection-department-filter"
                  options={departmentFilterOptions}
                  value={queryDepartment}
                  onSelect={handleDepartmentFilterChange}
                />
                <FilterDropdown
                  id="collection-area-filter"
                  options={areaFilterOptions}
                  value={queryAreaFilter}
                  onSelect={handleAreaFilterChange}
                />
                <FilterDropdown
                  id="collection-sort-filter"
                  options={sortOptions}
                  value={querySort}
                  onSelect={(value) => handleSortChange(value as CollectionSort)}
                />
              </div>
            </div>
            </div>

            <div className={styles.collectionBody}>
              {documentsHubMode === "card" ? (
                <div className={styles.cardGrid}>
                  {collectionDocuments.map((item) => (
                    <article key={item.documentId} className={styles.docCard}>
                      <div className={styles.docCardHero}>
                        <span aria-hidden="true" className={`${styles.docCardOrb} ${styles.docCardOrbLeft}`} />
                        <span aria-hidden="true" className={`${styles.docCardOrb} ${styles.docCardOrbRight}`} />
                        <div className={styles.docCardHeroContent}>
                          <strong className={styles.docCardTitleChip}>{formatDocumentDisplayName(item, props.documentProfiles)}</strong>
                          <span className={styles.docCardStatusBadge}>
                            <span aria-hidden="true" className={styles.docCardStatusDot} />
                            {statusLabel(item.status)}
                          </span>
                        </div>
                      </div>
                      <button
                        type="button"
                        className={styles.docCardBodyButton}
                        onClick={() => handleRecentOpen(item)}
                      >
                        <div className={styles.docCardBody}>
                          <div className={styles.docCardDetails}>
                            <span><strong>Autor</strong>{userNameById.get(item.ownerId) ?? item.ownerId ?? "-"}</span>
                            <span><strong>Criado em</strong>{formatDateOnly(item.createdAt)}</span>
                            <span><strong>Versao</strong>v{item.profileSchemaVersion ?? 1}</span>
                          </div>
                          <div className={styles.docCardTags}>
                            <span
                              className={styles.metaChip}
                              style={
                                {
                                  ["--chip-color" as string]: departmentColorByCode[normalizeToken(item.department)] ?? "#6A5C62",
                                  ["--chip-soft" as string]: hexToRgba(departmentColorByCode[normalizeToken(item.department)] ?? "#6A5C62", 0.14),
                                } as React.CSSProperties
                              }
                            >
                              Depto: {item.department || "-"}
                            </span>
                            <span
                              className={styles.metaChipAccent}
                              style={
                                {
                                  ["--chip-color" as string]: areaColorByCode[normalizeAreaCode(item.processArea)] ?? "#6A5C62",
                                  ["--chip-soft" as string]: hexToRgba(areaColorByCode[normalizeAreaCode(item.processArea)] ?? "#6A5C62", 0.2),
                                } as React.CSSProperties
                              }
                            >
                              Área — {processAreaNameByCode.get(normalizeAreaCode(item.processArea)) ?? item.processArea ?? "Sem área"}
                            </span>
                          </div>
                          <div className={styles.docCardFooter}>
                            <span>{profileNameByCode.get(item.documentProfile) ?? item.documentProfile}</span>
                            <span>{item.expiryAt ? `Revisao: ${props.formatDate(item.expiryAt)}` : "Sem data de revisao"}</span>
                          </div>
                        </div>
                      </button>
                    </article>
                  ))}
                  {collectionDocuments.length === 0 && <div className={styles.emptyCard}>Nenhum documento encontrado.</div>}
                </div>
              ) : (
                <div className={styles.listTable}>
                  <div className={styles.listHeader}>
                    <span>Documento</span>
                    <span>Tipo</span>
                    <span>Status</span>
                    <span>Owner</span>
                    <span>Prox. revisao</span>
                  </div>
                  {collectionDocuments.map((item) => (
                    <button
                      key={item.documentId}
                      type="button"
                      className={styles.listRow}
                      onClick={() => handleRecentOpen(item)}
                    >
                      <span className={styles.listTitle}>
                        <strong>{formatDocumentDisplayName(item, props.documentProfiles)}</strong>
                        <small>
                          {item.documentCode || item.documentId} · {processAreaNameByCode.get(normalizeAreaCode(item.processArea)) ?? item.processArea ?? "Sem area"} · Autor: {userNameById.get(item.ownerId) ?? item.ownerId}
                        </small>
                      </span>
                      <span>{item.documentProfile.toUpperCase()}</span>
                      <span>{statusLabel(item.status)}</span>
                      <span>{item.department || "-"}</span>
                      <span>{item.expiryAt ? props.formatDate(item.expiryAt) : "-"}</span>
                    </button>
                  ))}
                  {collectionDocuments.length === 0 && <div className={styles.emptyCard}>Nenhum documento encontrado.</div>}
                </div>
              )}
            </div>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      {headerShell}

      <section className={styles.hub}>
        <section className={styles.dashboardKpiStrip}>
          <article className={styles.dashboardKpiItem}>
            <strong>{scopedDocuments.length}</strong>
            <small>Documentos Ativos</small>
          </article>
          <article className={styles.dashboardKpiItem}>
            <strong>{inReviewCount}</strong>
            <small>Em Revisao</small>
          </article>
          <article className={styles.dashboardKpiItem}>
            <strong>0</strong>
            <small>Notificacoes Pendentes</small>
          </article>
          <article className={styles.dashboardKpiItem}>
            <strong>{props.documentProfiles.length}</strong>
            <small>Profiles Disponiveis</small>
          </article>
        </section>

        <section className={styles.dashboardMainGrid}>
          <article className={`${styles.dashboardCard} ${styles.dashboardCardBlue}`}>
            <header className={styles.dashboardCardHeader}>
              <h2>Ultimos Documentos</h2>
            </header>
            <div className={styles.dashboardTimeline}>
              {latestDocuments.map((item, index) => (
                <button key={item.documentId} type="button" className={styles.dashboardTimelineItem} onClick={() => handleRecentOpen(item)}>
                  <span className={`${styles.dashboardTimelineDot} ${index === 0 ? styles.dashboardTimelineDotActive : ""}`} />
                  <div className={styles.dashboardTimelineContent}>
                    <strong>{formatDocumentDisplayName(item, props.documentProfiles)}</strong>
                    <small>{props.formatDate(item.createdAt)}</small>
                  </div>
                  <span className={styles.dashboardTimelineType}>
                    {profileNameByCode.get(item.documentProfile) ?? item.documentProfile}
                  </span>
                </button>
              ))}
              {latestDocuments.length === 0 ? <div className={styles.emptyCard}>Sem documentos recentes.</div> : null}
            </div>
          </article>

          <div className={styles.dashboardRightStack}>
            <article className={`${styles.dashboardCard} ${styles.dashboardCardOrange}`}>
              <header className={styles.dashboardCardHeader}>
                <h2>Pendencias de revisao</h2>
              </header>
              <div className={styles.dashboardPendingTable}>
                <div className={styles.dashboardPendingHead}>
                  <span>Prioridade</span>
                  <span>Revisao</span>
                </div>
                {["Prioritario", "Medio", "Represente", "Politico"].map((priority) => (
                  <div key={priority} className={styles.dashboardPendingRow}>
                    <span className={styles.dashboardPriorityTag}>{priority}</span>
                    <span>{inReviewCount > 0 ? `${inReviewCount} documento(s) aguardando revisao.` : "Nenhum documento aguardando revisao."}</span>
                  </div>
                ))}
              </div>
            </article>

            <article className={`${styles.dashboardCard} ${styles.dashboardCardRed}`}>
              <header className={styles.dashboardCardHeader}>
                <h2>Expiracoes Proximas</h2>
              </header>
              <div className={styles.dashboardExpiry}>
                <div className={styles.dashboardExpiryCount}>
                  <strong>{nextExpiryDays}</strong>
                  <small>Dias Restantes</small>
                </div>
                <p>{expiringSoon > 0 ? `${expiringSoon} documento(s) vencendo nos proximos 30 dias.` : "Nenhuma expiracao nos proximos 30 dias."}</p>
              </div>
            </article>
          </div>
        </section>

        <section className={styles.dashboardBottomGrid}>
          <article className={`${styles.dashboardCard} ${styles.dashboardCardGreen}`}>
            <header className={styles.dashboardCardHeader}>
              <h2>Snapshot Operacional</h2>
            </header>
            <div className={styles.dashboardSnapshot}>
              <span>Sem notificacoes nao lidas.</span>
            </div>
          </article>
          <article className={`${styles.dashboardCard} ${styles.dashboardCardPurple}`}>
            <header className={styles.dashboardCardHeader}>
              <h2>Foco Metal Nobre</h2>
            </header>
            <div className={styles.dashboardFocus}>
              <strong>{focusAreaCard ? `${focusAreaCard.label} (${focusAreaCard.count})` : "Sem area (0)"}</strong>
              <span>{focusAreaCard?.description ?? "Sem dados de foco operacional."}</span>
            </div>
          </article>
        </section>
      </section>
    </div>
  );
}










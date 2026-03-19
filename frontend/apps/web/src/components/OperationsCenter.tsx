import { metalNobreProcessAreaHint } from "../features/documents/adapters/metalNobreExperience";
import type { DocumentProfileItem, NotificationItem, ProcessAreaItem, SearchDocumentItem } from "../lib.types";
import { WorkspaceViewFrame } from "./WorkspaceViewFrame";

type OperationsCenterProps = {
  documents: SearchDocumentItem[];
  notifications: NotificationItem[];
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  formatDate: (value?: string) => string;
  onCreateDocument: () => void;
  onOpenDocument: (documentId: string) => void | Promise<void>;
};

export function OperationsCenter(props: OperationsCenterProps) {
  const profileNameByCode = new Map(props.documentProfiles.map((item) => [item.code, item.name]));
  const areaNameByCode = new Map(props.processAreas.map((item) => [item.code, item.name]));
  const pendingReviews = props.documents.filter((item) => item.status === "IN_REVIEW");
  const recentDocuments = [...props.documents]
    .sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime())
    .slice(0, 5);
  const unreadNotifications = props.notifications.filter((item) => item.status !== "READ");
  const expiringSoon = props.documents
    .filter((item) => {
      if (!item.expiryAt) return false;
      const diff = new Date(item.expiryAt).getTime() - Date.now();
      return diff > 0 && diff <= 1000 * 60 * 60 * 24 * 30;
    })
    .slice(0, 5);
  const areaCountByCode = new Map<string, number>();
  for (const document of props.documents) {
    const areaCode = (document.processArea ?? "").trim().toLowerCase();
    if (areaCode === "") {
      continue;
    }
    areaCountByCode.set(areaCode, (areaCountByCode.get(areaCode) ?? 0) + 1);
  }
  const processAreaSnapshot = Array.from(areaCountByCode.entries())
    .map(([code, count]) => ({ code, label: areaNameByCode.get(code) ?? code, count, hint: metalNobreProcessAreaHint(code) }))
    .sort((left, right) => right.count - left.count)
    .slice(0, 5);

  return (
    <WorkspaceViewFrame
      kicker="Centro operacional"
      title="Painel documental"
      description="Visao executiva do que pede atencao agora, sem depender de realtime obrigatorio para ser util no dia a dia."
      actions={<button type="button" onClick={props.onCreateDocument}>Criar novo documento</button>}
      stats={(
        <div className="catalog-stats">
          <article className="catalog-stat"><span>Documentos ativos</span><strong>{props.documents.length}</strong><small>Acervo indexado no workspace</small></article>
          <article className="catalog-stat"><span>Em revisao</span><strong>{pendingReviews.length}</strong><small>Fila operacional imediata</small></article>
          <article className="catalog-stat"><span>Notificacoes pendentes</span><strong>{unreadNotifications.length}</strong><small>Leitura do usuario autenticado</small></article>
          <article className="catalog-stat"><span>Profiles disponiveis</span><strong>{props.documentProfiles.length}</strong><small>Motor profile-first pronto para authoring</small></article>
        </div>
      )}
    >
      <div className="operations-grid">
        <section className="catalog-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Recentes</p>
              <h2>Ultimos documentos</h2>
            </div>
          </div>
          <ul className="catalog-mini-list">
            {recentDocuments.map((item) => (
              <li key={item.documentId}>
                <button type="button" className="inline-link-button" onClick={() => void props.onOpenDocument(item.documentId)}>{item.title}</button>
                <small>{profileNameByCode.get(item.documentProfile) ?? item.documentProfile} / {props.formatDate(item.createdAt)}</small>
              </li>
            ))}
            {recentDocuments.length === 0 && <li><span>Nenhum documento carregado.</span></li>}
          </ul>
        </section>

        <section className="catalog-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Approvals</p>
              <h2>Pendencias de revisao</h2>
            </div>
          </div>
          <ul className="catalog-mini-list">
            {pendingReviews.map((item) => (
              <li key={item.documentId}>
                <button type="button" className="inline-link-button" onClick={() => void props.onOpenDocument(item.documentId)}>{item.title}</button>
                <small>{item.processArea || "Sem area"} / {item.ownerId}</small>
              </li>
            ))}
            {pendingReviews.length === 0 && <li><span>Nenhum documento aguardando revisao.</span></li>}
          </ul>
        </section>

        <section className="catalog-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Notificacoes</p>
              <h2>Snapshot operacional</h2>
            </div>
          </div>
          <ul className="catalog-mini-list">
            {unreadNotifications.slice(0, 5).map((item) => (
              <li key={item.id}>
                <span>{item.title}</span>
                <small>{props.formatDate(item.createdAt)}</small>
              </li>
            ))}
            {unreadNotifications.length === 0 && <li><span>Sem notificacoes nao lidas.</span></li>}
          </ul>
        </section>

        <section className="catalog-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Cadencia</p>
              <h2>Expiracoes proximas</h2>
            </div>
          </div>
          <ul className="catalog-mini-list">
            {expiringSoon.map((item) => (
              <li key={item.documentId}>
                <button type="button" className="inline-link-button" onClick={() => void props.onOpenDocument(item.documentId)}>{item.title}</button>
                <small>Vence em {props.formatDate(item.expiryAt)}</small>
              </li>
            ))}
            {expiringSoon.length === 0 && <li><span>Nenhuma expiracao nos proximos 30 dias.</span></li>}
          </ul>
        </section>

        <section className="catalog-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Processos</p>
              <h2>Foco Metal Nobre</h2>
            </div>
          </div>
          <ul className="catalog-mini-list">
            {processAreaSnapshot.map((item) => (
              <li key={item.code}>
                <span>{item.label} ({item.count})</span>
                <small>{item.hint}</small>
              </li>
            ))}
            {processAreaSnapshot.length === 0 && <li><span>Sem processos com documentos no recorte atual.</span></li>}
          </ul>
        </section>
      </div>
    </WorkspaceViewFrame>
  );
}

import type { NotificationItem } from "../lib.types";

type NotificationsPanelProps = {
  notifications: NotificationItem[];
  formatDate: (value?: string) => string;
  onMarkRead: (notificationId: string) => void | Promise<void>;
};

export function NotificationsPanel(props: NotificationsPanelProps) {
  const unreadCount = props.notifications.filter((item) => item.status !== "READ").length;

  return (
    <section className="catalog-shell">
      <div className="catalog-header">
        <div>
          <p className="catalog-kicker">Operacao</p>
          <h1>Notificacoes</h1>
          <p>Fila operacional da sessao autenticada com leitura e priorizacao de eventos de workflow.</p>
        </div>
      </div>

      <div className="catalog-stats compact">
        <article className="catalog-stat">
          <span>Total</span>
          <strong>{props.notifications.length}</strong>
          <small>Notificacoes carregadas no workspace</small>
        </article>
        <article className="catalog-stat">
          <span>Pendentes</span>
          <strong>{unreadCount}</strong>
          <small>Aguardando leitura</small>
        </article>
      </div>

      <div className="catalog-grid single">
        <section className="catalog-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Inbox</p>
              <h2>Eventos recentes</h2>
            </div>
          </div>
          <ul className="catalog-mini-list">
            {props.notifications.map((item) => (
              <li key={item.id}>
                <div>
                  <strong>{item.title}</strong>
                  <p>{item.message}</p>
                  <small>{item.eventType} / {props.formatDate(item.createdAt)}</small>
                </div>
                <div className="stack">
                  <span>{item.status}</span>
                  {item.status !== "READ" && <button type="button" className="ghost-button" onClick={() => void props.onMarkRead(item.id)}>Marcar como lida</button>}
                </div>
              </li>
            ))}
            {props.notifications.length === 0 && <li><span>Nenhuma notificacao disponivel.</span></li>}
          </ul>
        </section>
      </div>
    </section>
  );
}

import type { NotificationItem } from "../lib.types";

type NotificationsPanelProps = {
  notifications: NotificationItem[];
  formatDate: (value?: string) => string;
  onMarkRead: (notificationId: string) => void | Promise<void>;
};

export function NotificationsPanel(props: NotificationsPanelProps) {
  return (
    <section className="panel">
      <div className="panel-heading"><p className="kicker">Operacao</p><h2>Notificacoes</h2></div>
      <ul className="mini-list">
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
  );
}

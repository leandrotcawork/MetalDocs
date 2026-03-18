import type { CurrentUser, NotificationItem, UserRole } from "../lib.types";

type AppShellHeaderProps = {
  user: CurrentUser;
  apiBaseUrl: string;
  currentUserRoles: UserRole[];
  notifications: NotificationItem[];
  onLogout: () => void | Promise<void>;
};

export function AppShellHeader(props: AppShellHeaderProps) {
  const pendingNotifications = props.notifications.filter((item) => item.status !== "READ").length;

  return (
    <header className="hero">
      <div>
        <p className="eyebrow">MetalDocs Control Room</p>
        <h1>Operacao documental profissional com identidade real.</h1>
        <p className="hero-copy">Usuario atual: {props.user.displayName} ({props.user.username}) - roles: {props.currentUserRoles.join(", ") || "sem role"}.</p>
      </div>
      <div className="hero-panel">
        <span>Runtime</span>
        <strong>{props.apiBaseUrl}</strong>
        <small>{pendingNotifications} notificacao(oes) pendentes</small>
        <button data-testid="logout-button" type="button" className="ghost-button" onClick={() => void props.onLogout()}>Logout</button>
      </div>
    </header>
  );
}

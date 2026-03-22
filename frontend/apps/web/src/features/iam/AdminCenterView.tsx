import { useEffect, useMemo } from "react";
import { ManagedUsersSection } from "../../components/ManagedUsersPanel";
import { WorkspaceDataState } from "../../components/WorkspaceDataState";
import { WorkspaceViewFrame } from "../../components/WorkspaceViewFrame";
import { useManagedUsers } from "./useManagedUsers";
import { useAdminCenter } from "./useAdminCenter";
import styles from "./AdminCenterView.module.css";

function formatDate(value?: string): string {
  if (!value) return "-";
  return new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" }).format(new Date(value));
}

export function AdminCenterView() {
  const adminCenter = useAdminCenter();
  const managedUsersApi = useManagedUsers(adminCenter.refresh);
  const selectedManagedUser = useMemo(
    () => managedUsersApi.managedUsers.find((item) => item.userId === managedUsersApi.managedUserForm.userId) ?? null,
    [managedUsersApi.managedUserForm.userId, managedUsersApi.managedUsers],
  );

  useEffect(() => {
    void adminCenter.refresh();
  }, [adminCenter.refresh]);

  useEffect(() => {
    if (!managedUsersApi.managedUserForm.userId) {
      return;
    }
    const current = managedUsersApi.managedUsers.find((item) => item.userId === managedUsersApi.managedUserForm.userId);
    if (!current) {
      return;
    }
    managedUsersApi.setManagedUserForm((previous) => ({
      ...previous,
      displayName: current.displayName,
      email: current.email ?? "",
      isActive: current.isActive,
      mustChangePassword: current.mustChangePassword,
      roles: Array.isArray(current.roles) && current.roles.length > 0 ? current.roles : previous.roles,
      resetPassword: "",
    }));
  }, [managedUsersApi.managedUserForm.userId, managedUsersApi.managedUsers, managedUsersApi.setManagedUserForm]);

  const onlineCount = adminCenter.onlineUsers.length;
  const latestActivity = adminCenter.recentActivities[0]?.occurredAt;

  return (
    <WorkspaceViewFrame
      testId="admin-center"
      kicker="IAM + Observabilidade"
      title="Central do Admin"
      description="Controle de usuarios, presenca online e ultimas atividades do sistema."
    >
      <div className={styles.shell}>
        <WorkspaceDataState
          loadState={adminCenter.loadState}
          isEmpty={adminCenter.users.length === 0}
          emptyTitle="Nenhum usuario interno cadastrado"
          emptyDescription="Crie o primeiro usuario para iniciar a administracao de acesso."
          loadingLabel="Atualizando base administrativa"
          onRetry={adminCenter.refresh}
        />

        <section className={styles.summary}>
          <div className={styles.summaryCard}>
            <span>Usuarios online</span>
            <strong>{onlineCount}</strong>
          </div>
          <div className={styles.summaryCard}>
            <span>Ultima atividade</span>
            <strong>{formatDate(latestActivity)}</strong>
          </div>
          <div className={styles.summaryCard}>
            <span>Total usuarios</span>
            <strong>{adminCenter.users.length}</strong>
          </div>
        </section>

        <section className={styles.grid}>
          <div className={styles.panel}>
            <div className={styles.panelHeader}>
              <div>
                <p className={styles.kicker}>Presenca</p>
                <h2 className={styles.panelTitle}>Usuarios online</h2>
              </div>
              <span className={styles.panelBadge}>{onlineCount} ativos</span>
            </div>
            {onlineCount === 0 ? (
              <p className={styles.empty}>Nenhum usuario online agora.</p>
            ) : (
              <ul className={styles.list}>
                {adminCenter.onlineUsers.map((item) => (
                  <li key={item.userId} className={styles.listItem}>
                    <div className={styles.listMeta}>
                      <strong className={styles.listTitle}>{item.displayName}</strong>
                      <small className={styles.listSub}>{item.username}</small>
                    </div>
                    <span className={styles.listTime}>{formatDate(item.lastSeenAt)}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className={styles.panel}>
            <div className={styles.panelHeader}>
              <div>
                <p className={styles.kicker}>Auditoria</p>
                <h2 className={styles.panelTitle}>Ultimas atividades</h2>
              </div>
              <span className={styles.panelBadge}>Ultimos 10</span>
            </div>
            {adminCenter.recentActivities.length === 0 ? (
              <p className={styles.empty}>Nenhuma atividade registrada.</p>
            ) : (
              <ul className={styles.list}>
                {adminCenter.recentActivities.map((item) => (
                  <li key={item.id} className={styles.listItem}>
                    <div className={styles.listMeta}>
                      <strong className={styles.listTitle}>{item.action}</strong>
                      <small className={styles.listSub}>{item.actorId} • {item.resourceType}</small>
                    </div>
                    <span className={styles.listTime}>{formatDate(item.occurredAt)}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </section>

        <ManagedUsersSection
          loadState={adminCenter.loadState}
          userForm={managedUsersApi.userForm}
          managedUserForm={managedUsersApi.managedUserForm}
          managedUsers={managedUsersApi.managedUsers}
          selectedManagedUser={selectedManagedUser}
          formatDate={formatDate}
          onRefreshWorkspace={adminCenter.refresh}
          onUserFormChange={managedUsersApi.setUserForm}
          onManagedUserFormChange={managedUsersApi.setManagedUserForm}
          onSubmitCreateUser={managedUsersApi.handleCreateUser}
          onSelectManagedUser={managedUsersApi.selectManagedUser}
          onToggleRole={managedUsersApi.toggleManagedUserRole}
          onSaveManagedUser={managedUsersApi.handleSaveManagedUser}
          onAdminResetPassword={managedUsersApi.handleAdminResetPassword}
          onUnlockManagedUser={managedUsersApi.handleUnlockManagedUser}
        />
      </div>
    </WorkspaceViewFrame>
  );
}

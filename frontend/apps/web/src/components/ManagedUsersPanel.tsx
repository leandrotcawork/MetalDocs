import type { ManagedUserItem, UserRole } from "../lib.types";
import { WorkspaceViewFrame } from "./WorkspaceViewFrame";
import { WorkspaceDataState } from "./WorkspaceDataState";
import { FilterDropdown, type SelectMenuOption } from "./ui/FilterDropdown";
import styles from "./ManagedUsersPanel.module.css";

type CreateUserForm = {
  userId: string;
  username: string;
  email: string;
  displayName: string;
  password: string;
  roles: UserRole[];
};

type ManagedUserForm = {
  userId: string;
  displayName: string;
  email: string;
  isActive: boolean;
  mustChangePassword: boolean;
  roles: UserRole[];
  resetPassword: string;
};

type ManagedUsersPanelProps = {
  loadState: "idle" | "loading" | "ready" | "error";
  userForm: CreateUserForm;
  managedUserForm: ManagedUserForm;
  managedUsers: ManagedUserItem[];
  selectedManagedUser: ManagedUserItem | null;
  formatDate: (value?: string) => string;
  onRefreshWorkspace: () => void | Promise<void>;
  onUserFormChange: (next: CreateUserForm) => void;
  onManagedUserFormChange: (next: ManagedUserForm) => void;
  onSubmitCreateUser: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
  onSelectManagedUser: (item: ManagedUserItem) => void;
  onToggleRole: (role: UserRole) => void;
  onSaveManagedUser: () => void | Promise<void>;
  onAdminResetPassword: () => void | Promise<void>;
  onUnlockManagedUser: () => void | Promise<void>;
};

export function ManagedUsersPanel(props: ManagedUsersPanelProps) {
  return (
    <WorkspaceViewFrame
      testId="managed-users-panel"
      kicker="IAM + Auth"
      title="Usuarios internos"
      description="Administracao operacional de identidades internas, roles, estado de acesso e recuperacao de senha."
    >
      <ManagedUsersSection {...props} />
    </WorkspaceViewFrame>
  );
}

export function ManagedUsersSection(props: ManagedUsersPanelProps) {
  const roleOptions: SelectMenuOption[] = ["admin", "editor", "reviewer", "viewer"].map((role) => ({
    value: role,
    label: role,
  }));
  const selectedRole = props.selectedManagedUser?.roles?.[0] ?? "";

  const toInitials = (value: string) => {
    const [first, second] = value.trim().split(/\s+/);
    return [first?.[0], second?.[0]].filter(Boolean).join("").toUpperCase();
  };

  const roleChipClass = (role?: string) => {
    if (!role) return styles.roleChip;
    if (role === "admin") return `${styles.roleChip} ${styles.roleAdmin}`;
    if (role === "editor") return `${styles.roleChip} ${styles.roleEditor}`;
    return `${styles.roleChip} ${styles.roleViewer}`;
  };

  return (
    <>
      <WorkspaceDataState
        loadState={props.loadState}
        isEmpty={props.managedUsers.length === 0}
        emptyTitle="Nenhum usuario interno cadastrado"
        emptyDescription="Crie o primeiro usuario para iniciar a administracao de acesso."
        loadingLabel="Atualizando base de usuarios"
        onRetry={props.onRefreshWorkspace}
      />

      <div className={styles.sectionTitle}>Gestao de usuarios</div>
      <div className={styles.panelGrid}>
        <form data-testid="user-create-form" className={`${styles.panel} ${styles.stack}`} onSubmit={props.onSubmitCreateUser}>
          <div className={styles.panelHeader}>
            <div>
              <p className={styles.kicker}>Provisioning</p>
              <h2 className={styles.panelTitle}>Criar usuario</h2>
            </div>
          </div>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>userId opcional</span>
            <input data-testid="user-id" placeholder="userId opcional" value={props.userForm.userId} onChange={(event) => props.onUserFormChange({ ...props.userForm, userId: event.target.value })} />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>username</span>
            <input data-testid="user-username" placeholder="username" value={props.userForm.username} onChange={(event) => props.onUserFormChange({ ...props.userForm, username: event.target.value })} required />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>email</span>
            <input data-testid="user-email" placeholder="email" value={props.userForm.email} onChange={(event) => props.onUserFormChange({ ...props.userForm, email: event.target.value })} />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>display name</span>
            <input data-testid="user-display-name" placeholder="display name" value={props.userForm.displayName} onChange={(event) => props.onUserFormChange({ ...props.userForm, displayName: event.target.value })} required />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>senha inicial</span>
            <input data-testid="user-password" type="password" placeholder="senha inicial" value={props.userForm.password} onChange={(event) => props.onUserFormChange({ ...props.userForm, password: event.target.value })} required />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>role</span>
            <FilterDropdown
              id="user-role"
              value={props.userForm.roles[0]}
              options={roleOptions}
              onSelect={(value) => props.onUserFormChange({ ...props.userForm, roles: [value as UserRole] })}
            />
          </label>
          <button data-testid="user-submit" type="submit">Criar usuario</button>
        </form>
        <section className={styles.panel}>
          <div className={styles.panelHeader}>
            <div>
              <p className={styles.kicker}>Directory</p>
              <h2 className={styles.panelTitle}>Base de usuarios</h2>
            </div>
            <span className={styles.panelMeta}>{props.managedUsers.length} total</span>
          </div>
          <ul className={styles.list}>
            {props.managedUsers.map((item) => (
              <li key={item.userId} className={styles.listItem} onClick={() => props.onSelectManagedUser(item)}>
                <span className={styles.avatar}>{toInitials(item.displayName)}</span>
                <div className={styles.listMeta}>
                  <strong className={styles.listTitle}>{item.displayName}</strong>
                  <p className={styles.listSub}>{item.username}</p>
                  <small className={styles.listSub}>
                    {item.isActive ? "Ativo" : "Inativo"} / {item.mustChangePassword ? "troca obrigatoria" : "senha OK"} / falhas: {item.failedLoginAttempts}
                    {item.lastLoginAt ? ` / ultimo login: ${props.formatDate(item.lastLoginAt)}` : ""}
                  </small>
                </div>
                <span className={roleChipClass(item.roles?.[0])}>{item.roles?.[0] ?? "viewer"}</span>
              </li>
            ))}
          </ul>
        </section>
        <section className={`${styles.panel} ${styles.stack}`}>
          <div className={styles.panelHeader}>
            <div>
              <p className={styles.kicker}>Lifecycle</p>
              <h2 className={styles.panelTitle}>Editar usuario</h2>
            </div>
          </div>
          {!props.selectedManagedUser ? <p className={styles.hint}>Selecione um usuario da lista para editar estado operacional e roles.</p> : (
            <>
              <div className={styles.editHero}>
                <span className={styles.editAvatar}>{toInitials(props.selectedManagedUser.displayName)}</span>
                <div className={styles.editMeta}>
                  <span className={styles.editName}>{props.selectedManagedUser.displayName}</span>
                  <span className={styles.editSub}>{props.selectedManagedUser.username} • {selectedRole || "viewer"}</span>
                </div>
                <span className={styles.statusTag}>
                  <span className={styles.statusDot} />
                  {props.selectedManagedUser.isActive ? "Ativo" : "Inativo"}
                </span>
              </div>
              <p className={styles.hint}>Auth state atual: {props.selectedManagedUser.isActive ? "ativo" : "inativo"} / {props.selectedManagedUser.mustChangePassword ? "troca obrigatoria" : "senha estabilizada"} / falhas: {props.selectedManagedUser.failedLoginAttempts}</p>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>display name</span>
                <input value={props.managedUserForm.displayName} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, displayName: event.target.value })} placeholder="Display name" />
              </label>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>email</span>
                <input value={props.managedUserForm.email} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, email: event.target.value })} placeholder="Email" />
              </label>
              <label><input type="checkbox" checked={props.managedUserForm.isActive} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, isActive: event.target.checked })} /> Usuario ativo</label>
              <label><input type="checkbox" checked={props.managedUserForm.mustChangePassword} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, mustChangePassword: event.target.checked })} /> Exigir troca de senha</label>
              <div className={styles.detailSummary}>
                {(["admin", "editor", "reviewer", "viewer"] as UserRole[]).map((role) => <label key={role}><input type="checkbox" checked={props.managedUserForm.roles.includes(role)} onChange={() => props.onToggleRole(role)} /> {role}</label>)}
              </div>
              <button type="button" className={styles.buttonPrimary} onClick={() => void props.onSaveManagedUser()}>Salvar usuario</button>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>nova senha temporaria</span>
                <input type="password" value={props.managedUserForm.resetPassword} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, resetPassword: event.target.value })} placeholder="Nova senha temporaria" />
              </label>
              <div className={styles.actionRow}>
                <button type="button" className={styles.buttonWarn} onClick={() => void props.onAdminResetPassword()}>Resetar senha</button>
                <button type="button" className={styles.buttonOutline} onClick={() => void props.onUnlockManagedUser()}>Desbloquear acesso</button>
              </div>
            </>
          )}
        </section>
      </div>
    </>
  );
}

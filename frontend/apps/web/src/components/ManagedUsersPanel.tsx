import type { ManagedUserItem, UserRole } from "../lib.types";
import { WorkspaceViewFrame } from "./WorkspaceViewFrame";
import { WorkspaceDataState } from "./WorkspaceDataState";
import { FilterDropdown, type SelectMenuOption } from "./ui/FilterDropdown";

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

      <div className="catalog-grid">
        <form data-testid="user-create-form" className="catalog-panel stack" onSubmit={props.onSubmitCreateUser}>
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Provisioning</p>
              <h2>Criar usuario</h2>
            </div>
          </div>
          <input data-testid="user-id" placeholder="userId opcional" value={props.userForm.userId} onChange={(event) => props.onUserFormChange({ ...props.userForm, userId: event.target.value })} />
          <input data-testid="user-username" placeholder="username" value={props.userForm.username} onChange={(event) => props.onUserFormChange({ ...props.userForm, username: event.target.value })} required />
          <input data-testid="user-email" placeholder="email" value={props.userForm.email} onChange={(event) => props.onUserFormChange({ ...props.userForm, email: event.target.value })} />
          <input data-testid="user-display-name" placeholder="display name" value={props.userForm.displayName} onChange={(event) => props.onUserFormChange({ ...props.userForm, displayName: event.target.value })} required />
          <input data-testid="user-password" type="password" placeholder="senha inicial" value={props.userForm.password} onChange={(event) => props.onUserFormChange({ ...props.userForm, password: event.target.value })} required />
          <FilterDropdown
            id="user-role"
            value={props.userForm.roles[0]}
            options={roleOptions}
            onSelect={(value) => props.onUserFormChange({ ...props.userForm, roles: [value as UserRole] })}
          />
          <button data-testid="user-submit" type="submit">Criar usuario</button>
        </form>
        <section className="catalog-panel catalog-list-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Directory</p>
              <h2>Base de usuarios</h2>
            </div>
          </div>
          <ul className="catalog-mini-list">
            {props.managedUsers.map((item) => (
              <li key={item.userId} onClick={() => props.onSelectManagedUser(item)}>
                <div>
                  <strong>{item.displayName}</strong>
                  <p>{item.username} - {(Array.isArray(item.roles) ? item.roles : []).join(", ") || "sem role"}</p>
                  <small>{item.isActive ? "Ativo" : "Inativo"} / {item.mustChangePassword ? "troca obrigatoria" : "senha OK"} / falhas: {item.failedLoginAttempts}{item.lockedUntil ? ` / lock: ${props.formatDate(item.lockedUntil)}` : ""}{item.lastLoginAt ? ` / ultimo login: ${props.formatDate(item.lastLoginAt)}` : ""}</small>
                </div>
                <span>{item.userId}</span>
              </li>
            ))}
          </ul>
        </section>
        <section className="catalog-panel stack">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Lifecycle</p>
              <h2>Editar usuario</h2>
            </div>
          </div>
          {!props.selectedManagedUser ? <p className="hint">Selecione um usuario da lista para editar estado operacional e roles.</p> : (
            <>
              <p className="hint">Auth state atual: {props.selectedManagedUser.isActive ? "ativo" : "inativo"} / {props.selectedManagedUser.mustChangePassword ? "troca obrigatoria" : "senha estabilizada"} / falhas: {props.selectedManagedUser.failedLoginAttempts}</p>
              <input value={props.managedUserForm.displayName} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, displayName: event.target.value })} placeholder="Display name" />
              <input value={props.managedUserForm.email} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, email: event.target.value })} placeholder="Email" />
              <label><input type="checkbox" checked={props.managedUserForm.isActive} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, isActive: event.target.checked })} /> Usuario ativo</label>
              <label><input type="checkbox" checked={props.managedUserForm.mustChangePassword} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, mustChangePassword: event.target.checked })} /> Exigir troca de senha</label>
              <div className="detail-summary">
                {(["admin", "editor", "reviewer", "viewer"] as UserRole[]).map((role) => <label key={role}><input type="checkbox" checked={props.managedUserForm.roles.includes(role)} onChange={() => props.onToggleRole(role)} /> {role}</label>)}
              </div>
              <button type="button" className="ghost-button" onClick={() => void props.onSaveManagedUser()}>Salvar usuario</button>
              <input type="password" value={props.managedUserForm.resetPassword} onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, resetPassword: event.target.value })} placeholder="Nova senha temporaria" />
              <div className="detail-summary">
                <button type="button" className="ghost-button" onClick={() => void props.onAdminResetPassword()}>Resetar senha</button>
                <button type="button" className="ghost-button" onClick={() => void props.onUnlockManagedUser()}>Desbloquear acesso</button>
              </div>
            </>
          )}
        </section>
      </div>
    </>
  );
}

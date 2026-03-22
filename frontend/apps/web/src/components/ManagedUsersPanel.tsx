import { useMemo, useState } from "react";
import type { ManagedUserItem, UserRole } from "../lib.types";
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

interface ManagedUsersPanelProps {
  loadState: "idle" | "loading" | "ready" | "error";
  userForm: CreateUserForm;
  managedUserForm: ManagedUserForm;
  managedUsers: ManagedUserItem[];
  selectedManagedUser: ManagedUserItem | null;
  formatDate: (value?: string) => string;
  onRefreshWorkspace: () => void | Promise<void>;
  onUserFormChange: (next: CreateUserForm) => void;
  onManagedUserFormChange: (next: ManagedUserForm) => void;
  onCreateUser: () => void | Promise<void>;
  onSelectManagedUser: (item: ManagedUserItem) => void;
  onToggleRole: (role: UserRole) => void;
  onSaveManagedUser: () => void | Promise<void>;
  onAdminResetPassword: () => void | Promise<void>;
  onUnlockManagedUser: () => void | Promise<void>;
}

const PROFILE_OPTIONS: Array<{ value: UserRole; label: string }> = [
  { value: "admin", label: "Administrador" },
  { value: "editor", label: "Editor" },
  { value: "reviewer", label: "Revisor" },
  { value: "viewer", label: "Viewer" },
];

const DEPARTMENT_OPTIONS: SelectMenuOption[] = [
  { value: "Operacoes", label: "Operacoes" },
  { value: "Qualidade", label: "Qualidade" },
  { value: "Engenharia", label: "Engenharia" },
  { value: "Administrativo", label: "Administrativo" },
];

const PROCESS_AREA_OPTIONS: SelectMenuOption[] = [
  { value: "Administrativo", label: "Administrativo" },
  { value: "Producao", label: "Producao" },
  { value: "Logistica", label: "Logistica" },
  { value: "Suprimentos", label: "Suprimentos" },
];

function toInitials(value: string) {
  const [first, second] = value.trim().split(/\s+/);
  return [first?.[0], second?.[0]].filter(Boolean).join("").toUpperCase();
}

function roleLabel(role?: UserRole) {
  const match = PROFILE_OPTIONS.find((option) => option.value === role);
  return match?.label ?? "Viewer";
}

function roleChipClass(role?: UserRole) {
  if (role === "admin") return `${styles.roleChip} ${styles.roleChipAdmin}`;
  if (role === "editor") return `${styles.roleChip} ${styles.roleChipEditor}`;
  if (role === "reviewer") return `${styles.roleChip} ${styles.roleChipReviewer}`;
  return `${styles.roleChip} ${styles.roleChipViewer}`;
}

function departmentFromRole(role?: UserRole) {
  if (role === "admin") return "Operacoes";
  if (role === "editor") return "Qualidade";
  if (role === "reviewer") return "Engenharia";
  return "Administrativo";
}

export function ManagedUsersPanel(props: ManagedUsersPanelProps) {
  return <ManagedUsersSection {...props} />;
}

export function ManagedUsersSection(props: ManagedUsersPanelProps) {
  const [search, setSearch] = useState("");
  const [department, setDepartment] = useState("Operacoes");
  const [processArea, setProcessArea] = useState("Administrativo");
  const selectedRole = props.managedUserForm.roles[0] ?? "viewer";

  const filteredUsers = useMemo(() => {
    const query = search.trim().toLowerCase();
    const matches = !query
      ? props.managedUsers
      : props.managedUsers.filter((item) => item.displayName.toLowerCase().includes(query) || item.username.toLowerCase().includes(query));
    return matches.slice(0, 10);
  }, [props.managedUsers, search]);

  const handleCreateRoleChange = (value: string) => {
    props.onUserFormChange({
      ...props.userForm,
      roles: [value as UserRole],
    });
  };

  const handleManagedRoleChange = (value: string) => {
    props.onManagedUserFormChange({
      ...props.managedUserForm,
      roles: [value as UserRole],
    });
  };

  const handleDeactivateUser = async () => {
    props.onManagedUserFormChange({
      ...props.managedUserForm,
      isActive: false,
    });
    await props.onSaveManagedUser();
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

      <div className={styles.sectionTitle}>Gestao de Usuarios</div>

      <section className={styles.grid}>
        <article className={`${styles.card} ${styles.createCard}`}>
          <header className={styles.cardHeader}>
            <h3 className={styles.cardTitle}>Criar usuario</h3>
          </header>
          <div className={styles.cardBody}>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Nome completo</span>
              <input
                type="text"
                placeholder="ex: Joao da Silva"
                value={props.userForm.displayName}
                onChange={(event) => props.onUserFormChange({ ...props.userForm, displayName: event.target.value })}
              />
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Username</span>
              <input
                type="text"
                placeholder="ex: joao_silva"
                value={props.userForm.username}
                onChange={(event) => props.onUserFormChange({ ...props.userForm, username: event.target.value })}
              />
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>E-mail</span>
              <input
                type="email"
                placeholder="email@metalnobre.com"
                value={props.userForm.email}
                onChange={(event) => props.onUserFormChange({ ...props.userForm, email: event.target.value })}
              />
            </label>
            <div className={styles.inlineFields}>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>Departamento</span>
                <FilterDropdown
                  id="create-department"
                  value={department}
                  options={DEPARTMENT_OPTIONS}
                  onSelect={(value) => setDepartment(value)}
                />
              </label>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>Area de processo</span>
                <FilterDropdown
                  id="create-process-area"
                  value={processArea}
                  options={PROCESS_AREA_OPTIONS}
                  onSelect={(value) => setProcessArea(value)}
                />
              </label>
            </div>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Senha inicial</span>
              <input
                type="password"
                placeholder="senha temporaria"
                value={props.userForm.password}
                onChange={(event) => props.onUserFormChange({ ...props.userForm, password: event.target.value })}
              />
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Perfil</span>
              <select value={props.userForm.roles[0] ?? "viewer"} onChange={(event) => handleCreateRoleChange(event.target.value)}>
                {PROFILE_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
            <button type="button" className={`${styles.button} ${styles.buttonPrimary}`} onClick={() => void props.onCreateUser()}>
              + Criar usuario
            </button>
          </div>
        </article>

        <article className={`${styles.card} ${styles.baseCard}`}>
          <header className={styles.cardHeader}>
            <h3 className={styles.cardTitle}>Base de usuarios</h3>
            <span className={styles.cardMeta}>{props.managedUsers.length} total</span>
          </header>
          <div className={styles.searchWrap}>
            <input type="text" placeholder="Buscar usuario..." value={search} onChange={(event) => setSearch(event.target.value)} />
          </div>
          <ul className={styles.userList}>
            {filteredUsers.map((item) => (
              <li
                key={item.userId}
                className={`${styles.userRow} ${props.selectedManagedUser?.userId === item.userId ? styles.userRowSelected : ""}`}
                onClick={() => props.onSelectManagedUser(item)}
              >
                <span className={`${styles.avatar} ${item.isActive ? "" : styles.avatarInactive}`}>{toInitials(item.displayName)}</span>
                <span className={styles.userName}>{item.displayName}</span>
                <span className={roleChipClass(item.roles?.[0])}>{roleLabel(item.roles?.[0])}</span>
              </li>
            ))}
          </ul>
          <footer className={styles.listFooter}>
            Exibindo {filteredUsers.length} de {props.managedUsers.length} usuarios
          </footer>
        </article>

        <article className={`${styles.card} ${styles.editCard}`}>
          <header className={styles.cardHeader}>
            <h3 className={styles.cardTitle}>Editar usuario</h3>
          </header>

          {!props.selectedManagedUser ? (
            <div className={styles.emptyState}>Selecione um usuario na base para editar.</div>
          ) : (
            <>
              <div className={styles.editHero}>
                <span className={styles.heroAvatar}>{toInitials(props.selectedManagedUser.displayName).slice(0, 1)}</span>
                <div>
                  <p className={styles.heroName}>{props.selectedManagedUser.displayName}</p>
                  <p className={styles.heroSub}>
                    {props.selectedManagedUser.username} - {departmentFromRole(selectedRole)}
                  </p>
                </div>
                <span className={styles.statusTag}>
                  <span className={styles.statusDot} />
                  {props.managedUserForm.isActive ? "Ativo" : "Inativo"}
                </span>
              </div>

              <div className={styles.cardBody}>
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>Nome completo</span>
                  <input
                    type="text"
                    value={props.managedUserForm.displayName}
                    onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, displayName: event.target.value })}
                  />
                </label>

                <div className={styles.inlineFields}>
                  <label className={styles.field}>
                    <span className={styles.fieldLabel}>Departamento</span>
                    <input type="text" value={departmentFromRole(selectedRole)} readOnly />
                  </label>
                  <label className={styles.field}>
                    <span className={styles.fieldLabel}>Perfil</span>
                    <select value={selectedRole} onChange={(event) => handleManagedRoleChange(event.target.value)}>
                      {PROFILE_OPTIONS.map((option) => (
                        <option key={option.value} value={option.value}>
                          {option.label}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>
              </div>

              <div className={styles.actionsStack}>
                <button type="button" className={`${styles.button} ${styles.buttonPrimary}`} onClick={() => void props.onSaveManagedUser()}>
                  Salvar usuario
                </button>
                <div className={styles.divider} />
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>Nova senha temporaria</span>
                  <input
                    type="password"
                    placeholder="digite para resetar"
                    value={props.managedUserForm.resetPassword}
                    onChange={(event) => props.onManagedUserFormChange({ ...props.managedUserForm, resetPassword: event.target.value })}
                  />
                </label>
                <button type="button" className={`${styles.button} ${styles.buttonWarn}`} onClick={() => void props.onAdminResetPassword()}>
                  Resetar senha
                </button>
                <button type="button" className={`${styles.button} ${styles.buttonOutline}`} onClick={() => void props.onUnlockManagedUser()}>
                  Desbloquear acesso
                </button>
                <button type="button" className={`${styles.button} ${styles.buttonDanger}`} onClick={() => void handleDeactivateUser()}>
                  Desativar usuario
                </button>
              </div>
            </>
          )}
        </article>
      </section>
    </>
  );
}

export default ManagedUsersPanel;

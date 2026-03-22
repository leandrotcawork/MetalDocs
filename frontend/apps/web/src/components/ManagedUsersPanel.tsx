import { useEffect, useMemo, useRef, useState } from "react";
import type { ManagedUserItem, UserRole } from "../lib.types";
import { WorkspaceDataState } from "./WorkspaceDataState";
import type { SelectMenuOption } from "./ui/FilterDropdown";
import { DropdownFieldBox, TextFieldBox } from "./ui/FormFieldBox";
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
  const [editDepartment, setEditDepartment] = useState("Operacoes");
  const [editProcessArea, setEditProcessArea] = useState("Administrativo");
  const [syncedCardHeight, setSyncedCardHeight] = useState<number | null>(null);
  const editCardRef = useRef<HTMLElement | null>(null);
  const selectedRole = props.managedUserForm.roles[0] ?? "viewer";

  const filteredUsers = useMemo(() => {
    const query = search.trim().toLowerCase();
    return !query
      ? props.managedUsers
      : props.managedUsers.filter((item) => item.displayName.toLowerCase().includes(query) || item.username.toLowerCase().includes(query));
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

  useEffect(() => {
    if (!props.selectedManagedUser) return;
    const roleDepartment = departmentFromRole(props.selectedManagedUser.roles?.[0]);
    setEditDepartment(roleDepartment);
    setEditProcessArea(roleDepartment);
  }, [props.selectedManagedUser]);

  useEffect(() => {
    const editCard = editCardRef.current;
    if (!editCard) return;

    const updateHeight = () => {
      setSyncedCardHeight(Math.ceil(editCard.getBoundingClientRect().height));
    };

    updateHeight();

    if (typeof ResizeObserver === "undefined") return;

    const observer = new ResizeObserver(() => {
      updateHeight();
    });

    observer.observe(editCard);
    return () => observer.disconnect();
  }, []);

  return (
    <>
      {props.loadState !== "loading" && (
        <WorkspaceDataState
          loadState={props.loadState}
          isEmpty={props.managedUsers.length === 0}
          emptyTitle="Nenhum usuario interno cadastrado"
          emptyDescription="Crie o primeiro usuario para iniciar a administracao de acesso."
          loadingLabel="Atualizando base de usuarios"
          onRetry={props.onRefreshWorkspace}
        />
      )}

      <div className={styles.sectionTitle}>Gestao de Usuarios</div>

      <section className={styles.grid}>
        <article
          className={`${styles.card} ${styles.createCard}`}
          style={syncedCardHeight ? { height: `${syncedCardHeight}px` } : undefined}
        >
          <header className={`${styles.cardHeader} ${styles.createHeader}`}>
            <h3 className={styles.cardTitle}>Criar usuario</h3>
          </header>
          <div className={styles.cardBody}>
            <TextFieldBox
              id="create-full-name"
              label="Nome completo"
              placeholder="ex: Joao da Silva"
              value={props.userForm.displayName}
              onChange={(value) => props.onUserFormChange({ ...props.userForm, displayName: value })}
            />
            <TextFieldBox
              id="create-username"
              label="Username"
              placeholder="ex: joao_silva"
              value={props.userForm.username}
              onChange={(value) => props.onUserFormChange({ ...props.userForm, username: value })}
            />
            <TextFieldBox
              id="create-email"
              label="E-mail"
              type="email"
              placeholder="email@metalnobre.com"
              value={props.userForm.email}
              onChange={(value) => props.onUserFormChange({ ...props.userForm, email: value })}
            />
            <div className={styles.taxonomyFields}>
              <DropdownFieldBox
                id="create-department"
                label="Departamento"
                value={department}
                options={DEPARTMENT_OPTIONS}
                onSelect={(value) => setDepartment(value)}
              />
              <DropdownFieldBox
                id="create-process-area"
                label="Area de processo"
                value={processArea}
                options={PROCESS_AREA_OPTIONS}
                onSelect={(value) => setProcessArea(value)}
              />
            </div>
            <TextFieldBox
              id="create-password"
              label="Senha inicial"
              type="password"
              placeholder="senha temporaria"
              value={props.userForm.password}
              onChange={(value) => props.onUserFormChange({ ...props.userForm, password: value })}
            />
            <DropdownFieldBox
              id="create-profile"
              label="Perfil"
              value={props.userForm.roles[0] ?? "viewer"}
              options={PROFILE_OPTIONS.map((option) => ({ value: option.value, label: option.label }))}
              onSelect={handleCreateRoleChange}
            />
            <button type="button" className={`${styles.button} ${styles.buttonPrimary} ${styles.createButton}`} onClick={() => void props.onCreateUser()}>
              Criar usuario
            </button>
          </div>
        </article>

        <div
          className={`${styles.card} ${styles.baseCard}`}
          style={syncedCardHeight ? { height: `${syncedCardHeight}px` } : undefined}
        >
          <div className={`${styles.cardHeader} ${styles.baseHeader}`}>
            <div className={styles.baseHeaderText}>
              <h3 className={styles.cardTitle}>Base de usuarios</h3>
            </div>
            <div className={styles.baseHeaderActions}>
              <span className={styles.cardMeta}>{props.managedUsers.length} total</span>
            </div>
          </div>
          <div className={styles.searchWrap}>
            <TextFieldBox
              id="users-search"
              placeholder="Buscar usuario..."
              type="search"
              value={search}
              onChange={setSearch}
            />
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
        </div>

        <article ref={editCardRef} className={`${styles.card} ${styles.editCard}`}>
          <header className={`${styles.cardHeader} ${styles.editHeader}`}>
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
                <TextFieldBox
                  id="edit-full-name"
                  label="Nome completo"
                  value={props.managedUserForm.displayName}
                  onChange={(value) => props.onManagedUserFormChange({ ...props.managedUserForm, displayName: value })}
                />

                <div className={styles.editMetaFields}>
                  <div className={styles.editMetaRow}>
                    <DropdownFieldBox
                      id="edit-department"
                      label="Departamento"
                      value={editDepartment}
                      options={DEPARTMENT_OPTIONS}
                      onSelect={setEditDepartment}
                    />
                    <DropdownFieldBox
                      id="edit-process-area"
                      label="Area de operacoes"
                      value={editProcessArea}
                      options={PROCESS_AREA_OPTIONS}
                      onSelect={setEditProcessArea}
                    />
                  </div>
                  <DropdownFieldBox
                    id="edit-profile"
                    label="Perfil"
                    value={selectedRole}
                    options={PROFILE_OPTIONS.map((option) => ({ value: option.value, label: option.label }))}
                    onSelect={handleManagedRoleChange}
                  />
                </div>
              </div>

              <div className={styles.actionsStack}>
                <button type="button" className={`${styles.button} ${styles.buttonPrimary}`} onClick={() => void props.onSaveManagedUser()}>
                  Salvar usuario
                </button>
                <div className={styles.divider} />
                <TextFieldBox
                  id="edit-reset-password"
                  label="Nova senha temporaria"
                  type="password"
                  placeholder="digite para resetar"
                  value={props.managedUserForm.resetPassword}
                  onChange={(value) => props.onManagedUserFormChange({ ...props.managedUserForm, resetPassword: value })}
                />
                <div className={styles.actionsRow}>
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
              </div>
            </>
          )}
        </article>
      </section>
    </>
  );
}

export default ManagedUsersPanel;

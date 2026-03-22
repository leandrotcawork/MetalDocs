# Padrões de Componentes — MetalDocs

## Estrutura de arquivos

```
frontend/apps/web/src/
├── pages/
│   └── admin/
│       ├── AdminPage.tsx
│       ├── AdminPage.module.css
│       └── sections/
│           ├── KpiSection.tsx
│           ├── KpiSection.module.css
│           ├── UserListSection.tsx
│           └── UserListSection.module.css
├── components/
│   ├── ui/
│   │   ├── Button/
│   │   │   ├── Button.tsx
│   │   │   ├── Button.module.css
│   │   │   └── index.ts
│   │   ├── Badge/
│   │   ├── Avatar/
│   │   ├── StatusPip/
│   │   └── GlassCard/
│   └── layout/
│       ├── AppHeader/
│       ├── AppSidebar/
│       └── PageShell/
├── hooks/
│   ├── useOnlineUsers.ts
│   └── useActivityFeed.ts
└── styles/
    └── globals.css   ← fonte de todas as variáveis CSS
```

---

## Anatomia de um componente

### `ComponentName.tsx` — estrutura base

```tsx
import { useState } from 'react'
import styles from './ComponentName.module.css'

interface Props {
  title: string
  count?: number
  variant?: 'default' | 'highlighted' | 'muted'
  onAction?: (id: string) => void
}

export default function ComponentName({
  title,
  count = 0,
  variant = 'default',
  onAction,
}: Props) {
  const [isExpanded, setIsExpanded] = useState(false)

  return (
    <div className={`${styles.root} ${styles[variant]}`}>
      <header className={styles.header}>
        <h3 className={styles.title}>{title}</h3>
        {count > 0 && (
          <span className={styles.badge}>{count}</span>
        )}
      </header>
    </div>
  )
}
```

### Regras de nomenclatura

| Elemento | Convenção | Exemplo |
|---|---|---|
| Arquivo componente | PascalCase | `UserCard.tsx` |
| Arquivo CSS | mesmo nome | `UserCard.module.css` |
| Prop de callback | prefixo `on` | `onDelete`, `onSave` |
| Prop booleana | prefixo `is`/`has` | `isLoading`, `hasError` |
| Classe root | sempre `.root` | `.root { ... }` |
| Variante | modificador | `.root.highlighted { ... }` |
| Elemento filho | descritivo | `.header`, `.title`, `.body` |

---

## `ComponentName.module.css` — estrutura base

```css
/* ─── Root ─── */
.root {
  background: var(--glass-1);
  backdrop-filter: blur(20px) saturate(1.2);
  -webkit-backdrop-filter: blur(20px) saturate(1.2);
  border: 1px solid var(--glass-border);
  border-radius: var(--radius-md);
  position: relative;
  overflow: hidden;
  transition: border-color var(--duration-normal), transform var(--duration-normal);
  animation: fadeInUp var(--duration-enter) var(--ease-out) both;
}

/* Linha de luz superior — efeito glass */
.root::before {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(255, 255, 255, 0.9) 40%,
    rgba(255, 255, 255, 1) 50%,
    rgba(255, 255, 255, 0.9) 60%,
    transparent 100%
  );
  pointer-events: none;
}

.root:hover {
  border-color: var(--glass-border-bright);
  transform: translateY(-1px);
}

/* ─── Variantes ─── */
.root.highlighted {
  border-color: rgba(196, 32, 32, 0.22);
  background: var(--crimson-soft);
}

/* ─── Header ─── */
.header {
  padding: var(--padding-card);
  border-bottom: 1px solid rgba(150, 80, 60, 0.10);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.title {
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  letter-spacing: var(--tracking-label);
  text-transform: uppercase;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

/* ─── Badge ─── */
.badge {
  font-size: var(--text-xs);
  font-weight: var(--weight-medium);
  padding: 3px 9px;
  border-radius: var(--radius-xs);
  letter-spacing: var(--tracking-wide);
  background: var(--glass-2);
  color: var(--text-ghost);
  border: 1px solid var(--glass-border);
}
```

---

## Padrões de estado

### Loading (skeleton)

```tsx
// No componente:
if (isLoading) return <ComponentNameSkeleton />

// ComponentNameSkeleton.tsx
function ComponentNameSkeleton() {
  return (
    <div className={styles.skeleton}>
      <div className={styles.skeletonLine} />
      <div className={styles.skeletonLine} style={{ width: '60%' }} />
    </div>
  )
}
```

```css
/* No .module.css */
.skeleton {
  padding: var(--padding-card);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.skeletonLine {
  height: 14px;
  border-radius: var(--radius-xs);
  background: linear-gradient(
    90deg,
    rgba(150, 80, 60, 0.06) 25%,
    rgba(150, 80, 60, 0.12) 50%,
    rgba(150, 80, 60, 0.06) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
```

### Empty state

```tsx
function EmptyState({ message }: { message: string }) {
  return (
    <div className={styles.emptyState}>
      <span className={styles.emptyIcon}>—</span>
      <p className={styles.emptyMessage}>{message}</p>
    </div>
  )
}
```

```css
.emptyState {
  padding: 2rem var(--padding-card);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  text-align: center;
}

.emptyIcon {
  font-family: var(--font-display);
  font-size: 28px;
  font-weight: 300;
  color: var(--text-ghost);
}

.emptyMessage {
  font-size: var(--text-sm);
  color: var(--text-ghost);
  font-weight: var(--weight-light);
  letter-spacing: var(--tracking-normal);
}
```

### Error state com retry

```tsx
function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className={styles.errorState}>
      <p className={styles.errorMessage}>{message}</p>
      <button className={styles.retryButton} onClick={onRetry}>
        Tentar novamente
      </button>
    </div>
  )
}
```

---

## Padrões de formulário controlado

```tsx
interface FormData {
  name: string
  email: string
  department: string
}

function UserForm({ onSubmit }: { onSubmit: (data: FormData) => void }) {
  const [form, setForm] = useState<FormData>({
    name: '',
    email: '',
    department: '',
  })

  const handleChange = (field: keyof FormData) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
      setForm(prev => ({ ...prev, [field]: e.target.value }))
    }

  return (
    <div className={styles.form}>
      <div className={styles.field}>
        <label htmlFor="name" className={styles.label}>Nome completo</label>
        <input
          id="name"
          type="text"
          className={styles.input}
          value={form.name}
          onChange={handleChange('name')}
          placeholder="ex: João da Silva"
        />
      </div>
      <button
        className={styles.submitButton}
        onClick={() => onSubmit(form)}
        disabled={!form.name || !form.email}
      >
        Criar usuário
      </button>
    </div>
  )
}
```

---

## Button variants

```tsx
// variants: 'primary' | 'ghost' | 'warn' | 'danger'
interface ButtonProps {
  children: React.ReactNode
  variant?: 'primary' | 'ghost' | 'warn' | 'danger'
  size?: 'sm' | 'md'
  fullWidth?: boolean
  disabled?: boolean
  onClick?: () => void
}
```

```css
.button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: 10px 18px;
  font-family: var(--font-body);
  font-size: var(--text-base);
  font-weight: var(--weight-medium);
  letter-spacing: var(--tracking-normal);
  cursor: pointer;
  border-radius: var(--radius-sm);
  border: none;
  transition: all var(--duration-normal);
}

.button.primary {
  background: linear-gradient(135deg, var(--crimson-mid), var(--crimson-bright));
  color: #fff;
  border: 1px solid rgba(196, 32, 32, 0.5);
  box-shadow: var(--shadow-crimson);
}

.button.primary:hover {
  box-shadow: var(--shadow-crimson-lg);
  transform: translateY(-1px);
}

.button.ghost {
  background: var(--glass-1);
  color: var(--text-secondary);
  border: 1px solid var(--glass-border);
}

.button.ghost:hover {
  background: rgba(255, 255, 255, 0.85);
  border-color: var(--glass-border-bright);
  color: var(--text-primary);
}

.button.warn {
  background: var(--color-warning-soft);
  color: var(--color-warning);
  border: 1px solid rgba(154, 100, 0, 0.22);
}

.button.danger {
  background: var(--color-danger-soft);
  color: var(--color-danger);
  border: 1px solid rgba(192, 48, 48, 0.18);
}

.button.fullWidth { width: 100%; }

.button.sm { padding: 7px 12px; font-size: var(--text-sm); }

.button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  transform: none !important;
}
```

---

## Avatar

```tsx
interface AvatarProps {
  initials: string
  size?: 'sm' | 'md' | 'lg'
  online?: boolean
}
```

```css
.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  background: linear-gradient(135deg, var(--crimson-soft), var(--gold-soft));
  border: 1px solid rgba(196, 32, 32, 0.25);
  font-family: var(--font-display);
  font-style: italic;
  color: rgba(200, 120, 80, 0.9);
  flex-shrink: 0;
  position: relative;
}

.avatar.sm  { width: 26px; height: 26px; font-size: 9px; border-radius: var(--radius-xs); }
.avatar.md  { width: 34px; height: 34px; font-size: 12px; }
.avatar.lg  { width: 46px; height: 46px; font-size: 18px; border-radius: var(--radius-md); }

.onlineRing {
  position: absolute;
  top: -2px; right: -2px;
  width: 9px; height: 9px;
  border-radius: 50%;
  background: var(--color-success);
  border: 2px solid var(--bg-elevated);
  box-shadow: 0 0 6px rgba(26, 122, 74, 0.6);
  animation: pipping 2.5s ease-in-out infinite;
}
```

---

## Stagger animation em listas

```tsx
// No componente de lista
{items.map((item, index) => (
  <UserItem
    key={item.id}
    {...item}
    style={{ animationDelay: `${index * 0.07}s` }}
  />
))}
```

```tsx
// No componente filho — aceitar style prop para o stagger
interface UserItemProps {
  name: string
  style?: React.CSSProperties
}

export default function UserItem({ name, style }: UserItemProps) {
  return (
    <div className={styles.item} style={style}>
      {name}
    </div>
  )
}
```

---

## Hook customizado padrão

```ts
// hooks/useOnlineUsers.ts
import { useState, useEffect } from 'react'

interface OnlineUser {
  id: string
  name: string
  username: string
  lastSeen: string
}

interface UseOnlineUsersResult {
  users: OnlineUser[]
  isLoading: boolean
  error: string | null
  refetch: () => void
}

export function useOnlineUsers(): UseOnlineUsersResult {
  const [users, setUsers] = useState<OnlineUser[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetch = async () => {
    setIsLoading(true)
    setError(null)
    try {
      const res = await window.fetch('/api/v1/users/online')
      if (!res.ok) throw new Error('Falha ao carregar usuários')
      const data = await res.json()
      setUsers(data.users)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Erro desconhecido')
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => { fetch() }, [])

  return { users, isLoading, error, refetch: fetch }
}
```

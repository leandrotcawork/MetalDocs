# Padrões de Componentes — MetalDocs Frontend

Cada seção abaixo é o padrão canônico de um componente. Copie e adapte — não reinvente.

---

## GlassCard

Wrapper base para todos os cards com efeito glass.

```tsx
// components/GlassCard/GlassCard.tsx
import styles from './GlassCard.module.css'
import { ReactNode } from 'react'

interface GlassCardProps {
  children: ReactNode
  className?: string
  hoverable?: boolean
}

export function GlassCard({ children, className, hoverable = true }: GlassCardProps) {
  return (
    <div className={[styles.card, hoverable && styles.hoverable, className].filter(Boolean).join(' ')}>
      {children}
    </div>
  )
}
```

```css
/* GlassCard.module.css */
.card {
  background: var(--color-glass-1);
  backdrop-filter: var(--backdrop-glass);
  -webkit-backdrop-filter: var(--backdrop-glass);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  position: relative;
  overflow: hidden;
  transition: border-color var(--transition-normal), transform var(--transition-normal);
}

/* Linha de luz no topo */
.card::before {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,0.9) 50%, transparent);
  pointer-events: none;
  z-index: 1;
}

.hoverable:hover {
  border-color: var(--color-border-strong);
  transform: translateY(-1px);
}
```

---

## KpiCard

```tsx
// components/KpiCard/KpiCard.tsx
import styles from './KpiCard.module.css'

type AccentColor = 'green' | 'amber' | 'crimson' | 'blue'

interface KpiCardProps {
  label: string
  value: string | number
  sub?: string
  delta?: string
  deltaType?: 'up' | 'neutral' | 'down'
  accent?: AccentColor
  icon?: ReactNode
}

export function KpiCard({ label, value, sub, delta, deltaType = 'neutral', accent = 'crimson', icon }: KpiCardProps) {
  return (
    <div className={[styles.card, styles[`accent_${accent}`]].join(' ')}>
      <div className={styles.accentLine} />
      <div className={styles.top}>
        {icon && <div className={[styles.iconWrap, styles[`icon_${accent}`]].join(' ')}>{icon}</div>}
        {delta && <span className={[styles.delta, styles[`delta_${deltaType}`]].join(' ')}>{delta}</span>}
      </div>
      <div className={styles.label}>{label}</div>
      <div className={styles.value}>{value}</div>
      {sub && <div className={styles.sub}>{sub}</div>}
    </div>
  )
}
```

```css
/* KpiCard.module.css */
.card {
  padding: var(--space-6);
  display: flex;
  flex-direction: column;
  position: relative;
  /* herda glass do GlassCard wrapper */
}

.accentLine {
  position: absolute;
  bottom: 0; left: var(--space-6); right: var(--space-6);
  height: 2px;
  border-radius: 2px 2px 0 0;
  opacity: 0.7;
}

.accent_green  .accentLine { background: linear-gradient(90deg, transparent, var(--color-success), transparent); }
.accent_amber  .accentLine { background: linear-gradient(90deg, transparent, var(--color-warning), transparent); }
.accent_crimson .accentLine { background: linear-gradient(90deg, transparent, var(--color-crimson-bright), transparent); }
.accent_blue   .accentLine { background: linear-gradient(90deg, transparent, var(--color-info), transparent); }

.top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--space-5);
}

.iconWrap {
  width: 38px; height: 38px;
  border-radius: var(--radius-sm);
  display: flex; align-items: center; justify-content: center;
  border: 1px solid;
  flex-shrink: 0;
}

.icon_green  { background: var(--color-success-soft);  border-color: var(--color-success-border); }
.icon_amber  { background: var(--color-warning-soft);  border-color: var(--color-warning-border); }
.icon_crimson { background: var(--color-crimson-soft); border-color: rgba(196,32,32,0.22); }
.icon_blue   { background: var(--color-info-soft);     border-color: var(--color-info-border); }

.delta {
  font-size: 10px;
  font-weight: 500;
  padding: 3px 9px;
  border-radius: var(--radius-full);
  letter-spacing: 0.04em;
}

.delta_up      { background: var(--color-success-soft);  color: var(--color-success);  border: 1px solid var(--color-success-border); }
.delta_down    { background: var(--color-danger-soft);   color: var(--color-danger);   border: 1px solid var(--color-danger-border); }
.delta_neutral { background: var(--color-glass-1);       color: var(--color-text-ghost); border: 1px solid var(--color-border); }

.label {
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--color-text-muted);
  margin-bottom: var(--space-1);
}

.value {
  font-family: var(--font-display);
  font-size: 46px;
  font-weight: 300;
  line-height: 1;
  color: var(--color-text-primary);
  letter-spacing: -0.01em;
}

.sub {
  font-size: 11px;
  color: var(--color-text-ghost);
  margin-top: var(--space-1);
  font-weight: 300;
}
```

---

## StatusBadge

```tsx
type BadgeVariant = 'success' | 'warning' | 'danger' | 'info' | 'default' | 'crimson'

interface StatusBadgeProps {
  label: string
  variant?: BadgeVariant
  dot?: boolean
}

export function StatusBadge({ label, variant = 'default', dot = false }: StatusBadgeProps) {
  return (
    <span className={[styles.badge, styles[variant]].join(' ')}>
      {dot && <span className={styles.dot} />}
      {label}
    </span>
  )
}
```

```css
.badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 3px 9px;
  border-radius: var(--radius-xs);
  border: 1px solid;
}

.success { background: var(--color-success-soft);  color: var(--color-success);  border-color: var(--color-success-border); }
.warning { background: var(--color-warning-soft);  color: var(--color-warning);  border-color: var(--color-warning-border); }
.danger  { background: var(--color-danger-soft);   color: var(--color-danger);   border-color: var(--color-danger-border); }
.info    { background: var(--color-info-soft);     color: var(--color-info);     border-color: var(--color-info-border); }
.crimson { background: var(--color-crimson-soft);  color: var(--color-crimson-bright); border-color: rgba(196,32,32,0.2); }
.default { background: var(--color-glass-1);       color: var(--color-text-muted); border-color: var(--color-border); }

.dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: currentColor;
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50%       { opacity: 0.4; }
}
```

---

## ActionButton

```tsx
type ButtonVariant = 'primary' | 'ghost' | 'warning' | 'danger' | 'success'
type ButtonSize    = 'sm' | 'md' | 'lg'

interface ActionButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  fullWidth?: boolean
  loading?: boolean
  icon?: ReactNode
}

export function ActionButton({
  variant = 'ghost', size = 'md', fullWidth, loading, icon, children, className, ...rest
}: ActionButtonProps) {
  return (
    <button
      className={[
        styles.btn,
        styles[variant],
        styles[size],
        fullWidth && styles.fullWidth,
        loading && styles.loading,
        className
      ].filter(Boolean).join(' ')}
      disabled={loading || rest.disabled}
      {...rest}
    >
      {icon && <span className={styles.icon}>{icon}</span>}
      {children}
    </button>
  )
}
```

```css
.btn {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-weight: 500;
  letter-spacing: 0.03em;
  cursor: pointer;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

/* Sizes */
.sm { font-size: 11px; padding: 6px 12px; }
.md { font-size: 12px; padding: 10px 18px; }
.lg { font-size: 14px; padding: 12px 24px; }

.fullWidth { width: 100%; justify-content: center; }

/* Variants */
.primary {
  background: linear-gradient(135deg, var(--color-crimson-mid), var(--color-crimson-bright));
  color: var(--color-text-inverse);
  border-color: rgba(196,32,32,0.5);
  box-shadow: var(--shadow-button);
}
.primary:hover:not(:disabled) {
  background: linear-gradient(135deg, var(--color-crimson-bright), #d42c2c);
  transform: translateY(-1px);
  box-shadow: 0 6px 24px rgba(122,18,18,0.35), inset 0 1px 0 rgba(255,255,255,0.12);
}

.ghost {
  background: rgba(255,255,255,0.6);
  color: var(--color-text-secondary);
  border-color: var(--color-border);
}
.ghost:hover:not(:disabled) { background: rgba(255,255,255,0.85); border-color: var(--color-border-strong); color: var(--color-text-primary); }

.warning { background: var(--color-warning-soft); color: var(--color-warning); border-color: var(--color-warning-border); }
.warning:hover:not(:disabled) { background: rgba(154,100,0,0.16); }

.danger { background: var(--color-danger-soft); color: var(--color-danger); border-color: var(--color-danger-border); }
.danger:hover:not(:disabled) { background: rgba(192,48,48,0.14); }

.success { background: var(--color-success-soft); color: var(--color-success); border-color: var(--color-success-border); }
.success:hover:not(:disabled) { background: rgba(26,122,74,0.16); }

.btn:disabled, .loading { opacity: 0.5; cursor: not-allowed; }

.icon { display: flex; align-items: center; font-size: 14px; }
```

---

## FormField

```tsx
interface FormFieldProps {
  label: string
  error?: string
  children: ReactNode
  required?: boolean
}

export function FormField({ label, error, children, required }: FormFieldProps) {
  return (
    <div className={[styles.field, error && styles.hasError].join(' ')}>
      <label className={styles.label}>
        {label}
        {required && <span className={styles.required}>*</span>}
      </label>
      {children}
      {error && <span className={styles.error}>{error}</span>}
    </div>
  )
}
```

```css
.field { display: flex; flex-direction: column; gap: var(--space-1); }

.label {
  font-size: 9px;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--color-text-ghost);
}

.required { color: var(--color-crimson-bright); margin-left: 2px; }

/* Inputs filhos via :global — exceção permitida */
.field :global(input),
.field :global(select),
.field :global(textarea) {
  background: rgba(255,255,255,0.7);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-primary);
  padding: 9px 12px;
  font-family: var(--font-body);
  font-size: 13px;
  font-weight: 300;
  outline: none;
  width: 100%;
  appearance: none;
  transition: all var(--transition-fast);
}

.field :global(input::placeholder),
.field :global(textarea::placeholder) { color: var(--color-text-ghost); }

.field :global(input:focus),
.field :global(select:focus),
.field :global(textarea:focus) {
  border-color: rgba(196,32,32,0.4);
  box-shadow: var(--shadow-focus);
  background: rgba(255,255,255,0.9);
}

.hasError :global(input),
.hasError :global(select) {
  border-color: var(--color-danger-border);
}

.error { font-size: 11px; color: var(--color-danger); }
```

---

## UserRow

```tsx
interface UserRowProps {
  name: string
  username: string
  role?: string
  time?: string
  online?: boolean
  selected?: boolean
  onClick?: () => void
}

export function UserRow({ name, username, role, time, online, selected, onClick }: UserRowProps) {
  const initials = name.split(' ').map(n => n[0]).slice(0, 2).join('')
  return (
    <div className={[styles.row, selected && styles.selected, onClick && styles.clickable].join(' ')} onClick={onClick}>
      <div className={styles.avatar}>
        {initials}
        {online && <span className={styles.onlineRing} />}
      </div>
      <div className={styles.info}>
        <span className={styles.name}>{name}</span>
        <span className={styles.username}>{username}{role && ` · ${role}`}</span>
      </div>
      {time && <span className={styles.time}>{time}</span>}
    </div>
  )
}
```

```css
.row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: 11px var(--space-5);
  border-bottom: 1px solid var(--color-border-divider);
  transition: background var(--transition-fast);
}

.row:last-child { border-bottom: none; }

.clickable { cursor: pointer; }
.clickable:hover { background: rgba(255,255,255,0.5); }

.selected {
  background: var(--color-crimson-soft);
  border-left: 2.5px solid rgba(196,32,32,0.6);
  padding-left: calc(var(--space-5) - 2.5px);
}

.avatar {
  width: 34px; height: 34px;
  border-radius: var(--radius-sm);
  background: linear-gradient(135deg, var(--color-crimson-soft), rgba(201,147,58,0.1));
  border: 1px solid rgba(196,32,32,0.25);
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 600;
  color: rgba(180,100,60,0.9);
  flex-shrink: 0;
  position: relative;
}

.onlineRing {
  position: absolute;
  top: -2px; right: -2px;
  width: 9px; height: 9px;
  border-radius: 50%;
  background: var(--color-success);
  border: 2px solid var(--color-bg-base);
  box-shadow: 0 0 6px rgba(26,122,74,0.5);
}

.info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
.name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.username { font-size: 10px; color: var(--color-text-ghost); font-weight: 300; }
.time { font-size: 11px; color: var(--color-text-ghost); white-space: nowrap; }
```

---

## ActivityItem

```tsx
type ActivityType = 'edit' | 'create' | 'login' | 'approve' | 'delete'

interface ActivityItemProps {
  type: ActivityType
  description: string
  actor: string
  timestamp: string
  resource?: string
}

const ACTIVITY_LABELS: Record<ActivityType, string> = {
  edit:    'EDIÇÃO',
  create:  'CRIAÇÃO',
  login:   'LOGIN',
  approve: 'APROVAÇÃO',
  delete:  'EXCLUSÃO',
}

export function ActivityItem({ type, description, actor, timestamp, resource }: ActivityItemProps) {
  return (
    <div className={styles.item}>
      <div className={[styles.iconWrap, styles[`icon_${type}`]].join(' ')}>
        {/* ícone SVG por tipo — veja assets/icons.tsx */}
      </div>
      <div className={styles.body}>
        <span className={styles.desc}>{description}</span>
        <span className={styles.meta}>{actor} · {timestamp}{resource && ` · ${resource}`}</span>
      </div>
      <span className={[styles.tag, styles[`tag_${type}`]].join(' ')}>{ACTIVITY_LABELS[type]}</span>
    </div>
  )
}
```

```css
.item {
  display: flex;
  gap: var(--space-3);
  padding: 11px var(--space-5);
  border-bottom: 1px solid var(--color-border-divider);
  align-items: flex-start;
  transition: background var(--transition-fast);
}
.item:last-child { border-bottom: none; }
.item:hover { background: rgba(255,255,255,0.5); }

.iconWrap {
  width: 28px; height: 28px;
  border-radius: var(--radius-xs);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
  margin-top: 2px;
  border: 1px solid;
}

.icon_edit    { background: var(--color-info-soft);    border-color: var(--color-info-border); }
.icon_create  { background: var(--color-success-soft); border-color: var(--color-success-border); }
.icon_login   { background: var(--color-warning-soft); border-color: var(--color-warning-border); }
.icon_approve { background: var(--color-crimson-soft); border-color: rgba(196,32,32,0.2); }
.icon_delete  { background: var(--color-danger-soft);  border-color: var(--color-danger-border); }

.body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.desc { font-size: 13px; color: var(--color-text-secondary); }
.meta { font-size: 11px; color: var(--color-text-ghost); font-weight: 300; }

.tag {
  font-size: 9px; font-weight: 600;
  letter-spacing: 0.08em;
  padding: 3px 8px;
  border-radius: var(--radius-xs);
  align-self: flex-start;
  flex-shrink: 0;
  margin-top: 2px;
  border: 1px solid;
}

.tag_edit    { background: var(--color-info-soft);    color: var(--color-info);    border-color: var(--color-info-border); }
.tag_create  { background: var(--color-success-soft); color: var(--color-success); border-color: var(--color-success-border); }
.tag_login   { background: var(--color-warning-soft); color: var(--color-warning); border-color: var(--color-warning-border); }
.tag_approve { background: var(--color-crimson-soft); color: var(--color-crimson-bright); border-color: rgba(196,32,32,0.2); }
.tag_delete  { background: var(--color-danger-soft);  color: var(--color-danger);  border-color: var(--color-danger-border); }
```

---

## SidebarNav

```tsx
interface NavItem {
  label: string
  href: string
  icon: ReactNode
  badge?: number
}

interface SidebarNavProps {
  sections: { label: string; items: NavItem[] }[]
  activePath: string
}

export function SidebarNav({ sections, activePath }: SidebarNavProps) {
  return (
    <nav className={styles.sidebar}>
      {sections.map(section => (
        <div key={section.label} className={styles.section}>
          <span className={styles.sectionLabel}>{section.label}</span>
          {section.items.map(item => (
            <a
              key={item.href}
              href={item.href}
              className={[styles.item, activePath === item.href && styles.active].join(' ')}
            >
              <span className={styles.itemIcon}>{item.icon}</span>
              {item.label}
              {item.badge != null && <span className={styles.badge}>{item.badge}</span>}
            </a>
          ))}
        </div>
      ))}
    </nav>
  )
}
```

```css
.sidebar {
  width: 230px;
  flex-shrink: 0;
  padding: var(--space-6) 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 2px;
  background: rgba(250,244,236,0.6);
  backdrop-filter: var(--backdrop-glass);
  -webkit-backdrop-filter: var(--backdrop-glass);
  border-right: 1px solid var(--color-border);
}

.section { margin-bottom: var(--space-6); }

.sectionLabel {
  display: block;
  font-size: 9px;
  font-weight: 500;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  color: var(--color-text-ghost);
  padding: var(--space-2) var(--space-3) var(--space-1);
  margin-bottom: 2px;
}

.item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 9px var(--space-3);
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 400;
  color: var(--color-text-muted);
  text-decoration: none;
  transition: all var(--transition-fast);
  position: relative;
  margin: 1px 0;
}

.item:hover { color: var(--color-text-secondary); background: rgba(255,255,255,0.55); }

.item.active {
  color: var(--color-crimson-bright);
  background: rgba(196,32,32,0.07);
  border: 1px solid rgba(196,32,32,0.16);
  font-weight: 500;
}

.item.active::before {
  content: '';
  position: absolute;
  left: 0; top: 20%; bottom: 20%;
  width: 2.5px;
  background: var(--color-crimson-bright);
  border-radius: 0 2px 2px 0;
  box-shadow: 0 0 8px rgba(196,32,32,0.4);
}

.itemIcon { display: flex; align-items: center; opacity: 0.8; }

.badge {
  margin-left: auto;
  font-size: 10px; font-weight: 500;
  background: var(--color-crimson-soft);
  color: rgba(196,32,32,0.85);
  border: 1px solid rgba(196,32,32,0.2);
  padding: 1px 7px;
  border-radius: var(--radius-full);
  min-width: 22px;
  text-align: center;
}
```

---

## PageHeader

```tsx
interface PageHeaderProps {
  title: string
  titleAccent?: string   // parte em itálico dourado
  subtitle?: string
  actions?: ReactNode
}

export function PageHeader({ title, titleAccent, subtitle, actions }: PageHeaderProps) {
  return (
    <header className={styles.header}>
      <div>
        <h1 className={styles.title}>
          {title}{' '}
          {titleAccent && <em className={styles.accent}>{titleAccent}</em>}
        </h1>
        {subtitle && <p className={styles.subtitle}>{subtitle}</p>}
      </div>
      {actions && <div className={styles.actions}>{actions}</div>}
    </header>
  )
}
```

```css
.header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: var(--space-8);
}

.title {
  font-family: var(--font-display);
  font-size: 42px;
  font-weight: 300;
  letter-spacing: 0.02em;
  line-height: 1;
  color: var(--color-text-primary);
}

.accent {
  font-style: italic;
  background: linear-gradient(135deg, #e8a070 0%, var(--color-gold) 40%, #c06030 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.subtitle {
  font-size: 12px;
  color: var(--color-text-ghost);
  margin-top: var(--space-2);
  font-weight: 300;
  letter-spacing: 0.04em;
}

.actions { display: flex; align-items: center; gap: var(--space-3); }
```

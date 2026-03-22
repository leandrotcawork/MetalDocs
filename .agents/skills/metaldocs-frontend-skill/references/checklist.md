# Checklist de Entrega + globals.css

## Checklist — verificar antes de entregar qualquer conversão

### TypeScript
- [ ] Nenhum `any` explícito
- [ ] Props tipadas com interface
- [ ] Retorno do componente tipado (implícito via JSX.Element está OK)
- [ ] `index.ts` de re-export criado
- [ ] Imports relativos corretos (`./`, `../`, `@/`)

### CSS Module
- [ ] Nenhum hex hardcodado — tudo via `var(--token)`
- [ ] Nenhum `!important`
- [ ] Nenhum seletor global (exceto `FormField` que usa `:global(input)` — padrão permitido)
- [ ] Classes em camelCase
- [ ] Sem CSS duplicado — reutilize classes compostas
- [ ] Animações declaradas uma única vez (não repetir keyframes em cada module)

### Design
- [ ] Glass com `backdrop-filter` e `-webkit-backdrop-filter`
- [ ] Linha de luz (`::before` linear-gradient) nos cards
- [ ] `transform: translateY(-1px)` no hover dos cards
- [ ] Tipografia usa `--font-display` para títulos, `--font-body` para corpo
- [ ] Títulos de página usam gradiente dourado no acento em itálico
- [ ] Status badges usam `StatusBadge` component — não reinventar
- [ ] `animation-delay` stagger incremental (0.07s) nas listas

### Estrutura de arquivos
- [ ] Componente em `src/components/NomeComponente/`
- [ ] Página em `src/pages/caminho/index.tsx`
- [ ] Tipos em `src/types/` (se tipo for compartilhado)
- [ ] Nada hardcodado que deveria vir de prop ou API

### Acessibilidade mínima
- [ ] `<button>` para ações clicáveis (não `<div onClick>`)
- [ ] `<a href>` para navegação
- [ ] `aria-hidden="true"` no mesh background decorativo
- [ ] `alt` em todas as `<img>`
- [ ] Labels associados a inputs

---

## globals.css — configuração base obrigatória

Cole isso em `frontend/apps/web/src/styles/globals.css`:

```css
/* Importar tokens primeiro */
@import './tokens.css';

/* Google Fonts */
@import url('https://fonts.googleapis.com/css2?family=Cormorant+Garamond:ital,wght@0,300;0,400;0,500;0,600;1,300;1,400&family=Outfit:wght@300;400;500;600&display=swap');

/* Reset */
*, *::before, *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html, body {
  height: 100%;
  overflow-x: hidden;
}

body {
  font-family: var(--font-body);
  font-size: 14px;
  font-weight: 400;
  line-height: 1.5;
  color: var(--color-text-primary);
  background: var(--color-bg-base);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* Tipografia base */
h1, h2, h3, h4, h5, h6 {
  font-family: var(--font-display);
  font-weight: 300;
  line-height: 1.1;
  color: var(--color-text-primary);
}

a {
  color: var(--color-crimson-bright);
  text-decoration: none;
  transition: color var(--transition-fast);
}
a:hover { color: var(--color-crimson-mid); }

button { cursor: pointer; }

/* Animações globais — referencie via composes ou className em modules */
@keyframes fadeInUp {
  from { transform: translateY(16px); opacity: 0; }
  to   { transform: translateY(0);    opacity: 1; }
}

@keyframes fadeInLeft {
  from { transform: translateX(-20px); opacity: 0; }
  to   { transform: translateX(0);     opacity: 1; }
}

@keyframes slideDown {
  from { transform: translateY(-100%); opacity: 0; }
  to   { transform: translateY(0);     opacity: 1; }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50%       { opacity: 0.4; }
}

@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Skeleton loading */
.skeleton {
  background: linear-gradient(
    90deg,
    rgba(200,170,150,0.12) 25%,
    rgba(200,170,150,0.22) 50%,
    rgba(200,170,150,0.12) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: var(--radius-sm);
}

/* Empty state */
.emptyState {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-10) var(--space-6);
  color: var(--color-text-ghost);
  font-size: 13px;
  font-weight: 300;
  text-align: center;
}

/* Scrollbar personalizada */
::-webkit-scrollbar { width: 5px; height: 5px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--color-border); border-radius: var(--radius-full); }
::-webkit-scrollbar-thumb:hover { background: var(--color-border-strong); }

/* Focus ring global */
:focus-visible {
  outline: 2px solid var(--color-crimson-bright);
  outline-offset: 2px;
  border-radius: var(--radius-xs);
}
```

---

## Estrutura de página padrão — Layout.tsx

```tsx
// components/Layout/Layout.tsx
import styles from './Layout.module.css'
import { Header } from '@/components/Header'
import { SidebarNav } from '@/components/SidebarNav'
import { ReactNode } from 'react'

interface LayoutProps {
  children: ReactNode
  activePath: string
}

export function Layout({ children, activePath }: LayoutProps) {
  return (
    <>
      <div className={styles.meshBg} aria-hidden="true">
        <div className={styles.orb1} />
        <div className={styles.orb2} />
        <div className={styles.orb3} />
      </div>

      <div className={styles.root}>
        <Header />
        <div className={styles.body}>
          <SidebarNav activePath={activePath} sections={NAV_SECTIONS} />
          <main className={styles.main}>{children}</main>
        </div>
      </div>
    </>
  )
}
```

```css
/* Layout.module.css */
.meshBg {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background:
    radial-gradient(ellipse 70% 60% at 15% 20%, rgba(210,140,100,0.22) 0%, transparent 60%),
    radial-gradient(ellipse 50% 50% at 85% 10%, rgba(200,160,120,0.16) 0%, transparent 55%),
    radial-gradient(ellipse 60% 70% at 70% 85%, rgba(180,100,80,0.14) 0%, transparent 60%),
    radial-gradient(ellipse 80% 40% at 50% 50%, rgba(255,240,220,0.12) 0%, transparent 70%);
  background-color: var(--color-bg-base);
}

.orb1, .orb2, .orb3 {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.6;
  animation: orbFloat linear infinite;
}

.orb1 {
  width: 600px; height: 600px;
  background: radial-gradient(circle, rgba(200,120,80,0.22) 0%, transparent 70%);
  top: -200px; left: -150px;
  animation-duration: 25s;
}

.orb2 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, rgba(220,170,110,0.18) 0%, transparent 70%);
  top: 10%; right: -100px;
  animation-duration: 32s;
  animation-direction: reverse;
}

.orb3 {
  width: 700px; height: 400px;
  background: radial-gradient(circle, rgba(201,147,58,0.15) 0%, transparent 70%);
  top: 40%; left: 40%;
  animation-duration: 20s;
  animation-delay: -5s;
}

@keyframes orbFloat {
  0%   { transform: translate(0, 0) scale(1); }
  25%  { transform: translate(30px, -20px) scale(1.05); }
  50%  { transform: translate(-20px, 30px) scale(0.97); }
  75%  { transform: translate(15px, 10px) scale(1.02); }
  100% { transform: translate(0, 0) scale(1); }
}

.root {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.body {
  display: flex;
  flex: 1;
}

.main {
  flex: 1;
  padding: var(--space-8) var(--space-8) var(--space-12);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}
```

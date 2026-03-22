# Glass Patterns — MetalDocs

Padrões de glassmorphism, backgrounds animados e efeitos visuais.

---

## Background animado (usado nas Pages)

O background é um componente próprio `<BgCanvas />` usado na raiz da page.

```tsx
// components/layout/BgCanvas/BgCanvas.tsx
export default function BgCanvas() {
  return (
    <div className={styles.canvas} aria-hidden="true">
      <div className={styles.orb1} />
      <div className={styles.orb2} />
      <div className={styles.orb3} />
      <div className={styles.orb4} />
    </div>
  )
}
```

```css
/* BgCanvas.module.css */
.canvas {
  position: fixed;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
}

.canvas::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse 70% 60% at 15% 20%, rgba(210, 140, 100, 0.18) 0%, transparent 60%),
    radial-gradient(ellipse 50% 50% at 85% 10%, rgba(200, 160, 120, 0.13) 0%, transparent 55%),
    radial-gradient(ellipse 60% 70% at 70% 85%, rgba(180, 100, 80, 0.11) 0%, transparent 60%),
    radial-gradient(ellipse 40% 40% at 30% 70%, rgba(230, 200, 170, 0.14) 0%, transparent 55%);
  background-color: var(--bg-base);
}

/* Noise overlay sutil */
.canvas::after {
  content: '';
  position: absolute;
  inset: 0;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
  opacity: 0.022;
  mix-blend-mode: multiply;
}

/* Orbs */
.orb1, .orb2, .orb3, .orb4 {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  animation: float linear infinite;
}

.orb1 {
  width: 600px; height: 600px;
  background: radial-gradient(circle, rgba(200, 120, 80, 0.18) 0%, transparent 70%);
  top: -200px; left: -150px;
  animation-duration: 25s;
}

.orb2 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, rgba(220, 170, 110, 0.14) 0%, transparent 70%);
  top: 10%; right: -100px;
  animation-duration: 32s;
  animation-direction: reverse;
}

.orb3 {
  width: 700px; height: 400px;
  background: radial-gradient(circle, rgba(180, 100, 70, 0.12) 0%, transparent 70%);
  bottom: -100px; left: 20%;
  animation-duration: 28s;
  animation-delay: -10s;
}

.orb4 {
  width: 300px; height: 300px;
  background: radial-gradient(circle, rgba(201, 147, 58, 0.12) 0%, transparent 70%);
  top: 40%; left: 40%;
  animation-duration: 20s;
  animation-delay: -5s;
}
```

---

## GlassCard — componente base reutilizável

```tsx
// components/ui/GlassCard/GlassCard.tsx
interface GlassCardProps {
  children: React.ReactNode
  className?: string  // exceção — permite composição com cx()
  animationDelay?: number
}

export default function GlassCard({ children, animationDelay }: GlassCardProps) {
  return (
    <div
      className={styles.card}
      style={animationDelay !== undefined
        ? { animationDelay: `${animationDelay}s` }
        : undefined}
    >
      {children}
    </div>
  )
}
```

```css
/* GlassCard.module.css */
.card {
  background: var(--glass-1);
  backdrop-filter: blur(20px) saturate(1.2);
  -webkit-backdrop-filter: blur(20px) saturate(1.2);
  border: 1px solid var(--glass-border);
  border-radius: var(--radius-md);
  position: relative;
  overflow: hidden;
  transition:
    border-color var(--duration-normal),
    transform var(--duration-normal);
  animation: fadeInUp var(--duration-enter) var(--ease-out) both;
}

/* Linha de luz superior */
.card::before {
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
  z-index: 1;
}

/* Inner glow */
.card::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: radial-gradient(
    ellipse 60% 40% at 50% 0%,
    rgba(255, 255, 255, 0.4) 0%,
    transparent 70%
  );
  pointer-events: none;
}

.card:hover {
  border-color: var(--glass-border-bright);
  transform: translateY(-1px);
}
```

---

## Header glass

```css
/* AppHeader.module.css */
.header {
  position: sticky;
  top: 0;
  z-index: var(--z-header);
  background: rgba(250, 245, 238, 0.75);
  backdrop-filter: blur(24px) saturate(1.4);
  -webkit-backdrop-filter: blur(24px) saturate(1.4);
  border-bottom: 1px solid var(--glass-border);
  animation: slideDown var(--duration-enter) var(--ease-out) both;
}
```

---

## Sidebar glass

```css
/* AppSidebar.module.css */
.sidebar {
  background: rgba(250, 244, 236, 0.6);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-right: 1px solid var(--glass-border);
  animation: fadeInLeft var(--duration-enter) var(--ease-out) 0.1s both;
}
```

---

## Status dot animado

```css
.statusDot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-success);
  animation: pipping 2s ease-in-out infinite;
}

/* Variantes de cor */
.statusDot.warning { background: var(--color-warning); }
.statusDot.danger  { background: var(--color-danger); }
.statusDot.info    { background: var(--color-info); }
```

---

## Accent line de cor em card (bottom border)

```css
/* Linha colorida na base do card indicando tipo */
.accentLine {
  position: absolute;
  bottom: 0; left: 1.5rem; right: 1.5rem;
  height: 2px;
  border-radius: 2px 2px 0 0;
  opacity: 0.7;
}

.accentLine.success {
  background: linear-gradient(90deg, transparent, var(--color-success), transparent);
}
.accentLine.warning {
  background: linear-gradient(90deg, transparent, var(--color-warning), transparent);
}
.accentLine.crimson {
  background: linear-gradient(90deg, transparent, var(--crimson-bright), transparent);
}
```

---

## Logo gem (SVG inline)

```tsx
// Usar como componente LogoGem.tsx
export default function LogoGem({ size = 40 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 40 40"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-label="MetalDocs logo"
    >
      <defs>
        <linearGradient id="gem-fill" x1="0" y1="0" x2="40" y2="40" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="rgba(160,24,24,0.9)" />
          <stop offset="60%" stopColor="rgba(110,14,14,0.85)" />
          <stop offset="100%" stopColor="rgba(60,8,8,0.9)" />
        </linearGradient>
      </defs>
      <polygon
        points="20,2 38,14 38,30 20,40 2,30 2,14"
        fill="url(#gem-fill)"
        stroke="rgba(220,120,80,0.5)"
        strokeWidth="0.8"
      />
      <polygon points="20,2 38,14 20,16 2,14" fill="rgba(255,200,160,0.12)" />
      <polygon points="2,14 20,16 20,40" fill="rgba(0,0,0,0.2)" />
      <polygon points="38,14 20,16 20,40" fill="rgba(255,200,160,0.06)" />
      <line x1="20" y1="2" x2="20" y2="16" stroke="rgba(255,200,160,0.3)" strokeWidth="0.5" />
      <line x1="2" y1="14" x2="38" y2="14" stroke="rgba(255,200,160,0.2)" strokeWidth="0.5" />
    </svg>
  )
}
```

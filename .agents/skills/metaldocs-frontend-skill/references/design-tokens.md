# Design Tokens — MetalDocs

Todas as variáveis CSS definidas em `globals.css`. **Nunca** usar valores literais nos `.module.css` — sempre referenciar estas variáveis.

---

## Paleta de cores

```css
/* Primário — Crimson */
--crimson: #7a1212;
--crimson-mid: #9e1818;
--crimson-bright: #c42020;
--crimson-soft: rgba(156, 24, 24, 0.08);
--crimson-glow: rgba(122, 18, 18, 0.18);

/* Acento — Gold */
--gold: #c9933a;
--gold-soft: rgba(201, 147, 58, 0.12);

/* Fundo — Warm White */
--bg-base: #faf7f3;
--bg-surface: #fdf9f5;
--bg-elevated: #ffffff;

/* Glass */
--glass-1: rgba(255, 255, 255, 0.72);
--glass-2: rgba(255, 255, 255, 0.55);
--glass-3: rgba(255, 255, 255, 0.35);
--glass-border: rgba(150, 80, 60, 0.13);
--glass-border-bright: rgba(150, 80, 60, 0.22);

/* Texto */
--text-primary: #1c1008;
--text-secondary: #3d2010;
--text-muted: #7a5540;
--text-ghost: #b09080;

/* Semânticas */
--color-success: #1a7a4a;
--color-success-soft: rgba(26, 122, 74, 0.10);
--color-warning: #9a6400;
--color-warning-soft: rgba(154, 100, 0, 0.10);
--color-info: #1a5a9a;
--color-info-soft: rgba(26, 90, 154, 0.09);
--color-danger: #c03030;
--color-danger-soft: rgba(192, 48, 48, 0.09);
```

---

## Tipografia

```css
/* Fontes — importadas via Google Fonts em index.html */
--font-display: 'Cormorant Garamond', Georgia, serif;
--font-body: 'Outfit', sans-serif;
--font-mono: 'Outfit', monospace; /* fallback — projeto não usa mono dedicada */

/* Escala */
--text-xs: 10px;
--text-sm: 11px;
--text-base: 13px;
--text-md: 15px;
--text-lg: 18px;
--text-xl: 22px;
--text-2xl: 28px;
--text-display: 42px;

/* Pesos */
--weight-light: 300;
--weight-regular: 400;
--weight-medium: 500;
--weight-semibold: 600;

/* Letter-spacing semântico */
--tracking-tight: 0.01em;
--tracking-normal: 0.02em;
--tracking-wide: 0.05em;
--tracking-wider: 0.08em;
--tracking-label: 0.10em;   /* labels uppercase */
--tracking-caps: 0.12em;    /* nav labels, badges */
```

---

## Espaçamento

```css
/* Base: 4px grid */
--space-1: 4px;
--space-2: 8px;
--space-3: 12px;
--space-4: 16px;
--space-5: 20px;
--space-6: 24px;
--space-8: 32px;
--space-10: 40px;
--space-12: 48px;
--space-16: 64px;

/* Padding de componentes */
--padding-card: 1.25rem 1.4rem;
--padding-card-sm: 1rem 1.2rem;
--padding-section: 2rem 2rem 3rem;
--padding-sidebar: 1.5rem 0.75rem;
```

---

## Border Radius

```css
--radius-xs: 6px;
--radius-sm: 10px;
--radius-md: 14px;
--radius-lg: 18px;
--radius-xl: 24px;
--radius-full: 9999px;
```

---

## Sombras

```css
--shadow-sm: 0 1px 3px rgba(30, 10, 5, 0.06);
--shadow-md: 0 4px 16px rgba(30, 10, 5, 0.08);
--shadow-lg: 0 8px 32px rgba(30, 10, 5, 0.10);
--shadow-crimson: 0 4px 16px rgba(122, 18, 18, 0.25);
--shadow-crimson-lg: 0 6px 24px rgba(122, 18, 18, 0.35);
```

---

## Animações

```css
/* Durações */
--duration-fast: 150ms;
--duration-normal: 200ms;
--duration-slow: 300ms;
--duration-enter: 400ms;

/* Easings */
--ease-out: cubic-bezier(0.16, 1, 0.3, 1);
--ease-in-out: cubic-bezier(0.45, 0, 0.55, 1);

/* Keyframes disponíveis globalmente */
/* fadeInUp: aparece de baixo para cima */
/* fadeInLeft: aparece da esquerda */
/* slideDown: desce do topo (header) */
/* pipping: pulse animado para status dots */
/* float: flutuação suave para orbs de fundo */
/* shimmer: loading skeleton */
```

### Uso padrão de animações de entrada

```css
/* Elemento único */
.card {
  animation: fadeInUp var(--duration-enter) var(--ease-out) both;
}

/* Lista com stagger — aplicar no .tsx via style prop */
/* No JSX: style={{ animationDelay: `${index * 0.07}s` }} */
.listItem {
  animation: fadeInUp var(--duration-enter) var(--ease-out) both;
}
```

---

## Z-index

```css
--z-base: 1;
--z-card: 10;
--z-dropdown: 50;
--z-sidebar: 100;
--z-header: 200;
--z-modal: 300;
--z-toast: 400;
```

---

## Breakpoints (media queries)

```css
/* Mobile first */
--bp-sm: 640px;
--bp-md: 768px;
--bp-lg: 1024px;
--bp-xl: 1280px;
--bp-2xl: 1440px;
```

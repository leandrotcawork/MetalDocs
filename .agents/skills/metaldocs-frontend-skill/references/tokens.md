# Design Tokens — MetalDocs Frontend

Este arquivo é a fonte da verdade para todos os tokens visuais do MetalDocs.
Cole o conteúdo abaixo em `frontend/apps/web/src/styles/tokens.css` e importe em `globals.css`.

## tokens.css completo

```css
:root {
  /* ─── CORES BASE ─── */
  --color-bg-base:         #faf7f3;   /* fundo da página */
  --color-bg-surface:      #ffffff;   /* cards, painéis */
  --color-bg-subtle:       #f5f0ea;   /* fundo alternativo, linhas zebra */

  /* ─── GLASS ─── */
  --color-glass-1:         rgba(255, 255, 255, 0.72);   /* card primário */
  --color-glass-2:         rgba(255, 255, 255, 0.55);   /* card secundário */
  --color-glass-3:         rgba(255, 255, 255, 0.35);   /* overlay sutil */

  /* ─── BORDAS ─── */
  --color-border:          rgba(150, 80,  60, 0.13);
  --color-border-strong:   rgba(150, 80,  60, 0.22);
  --color-border-divider:  rgba(150, 80,  60, 0.08);   /* separadores internos */

  /* ─── PALETA CRIMSON (brand) ─── */
  --color-crimson:         #7a1212;
  --color-crimson-mid:     #9e1818;
  --color-crimson-bright:  #c42020;
  --color-crimson-soft:    rgba(156,  24, 24, 0.08);
  --color-crimson-glow:    rgba(122,  18, 18, 0.18);

  /* ─── GOLD (acento) ─── */
  --color-gold:            #c9933a;
  --color-gold-soft:       rgba(201, 147, 58, 0.12);

  /* ─── TIPOGRAFIA ─── */
  --color-text-primary:    #1c1008;
  --color-text-secondary:  #3d2010;
  --color-text-muted:      #7a5540;
  --color-text-ghost:      #b09080;
  --color-text-inverse:    #ffffff;

  /* ─── STATUS ─── */
  --color-success:         #1a7a4a;
  --color-success-soft:    rgba( 26, 122, 74, 0.10);
  --color-success-border:  rgba( 26, 122, 74, 0.20);

  --color-warning:         #9a6400;
  --color-warning-soft:    rgba(154, 100,  0, 0.10);
  --color-warning-border:  rgba(154, 100,  0, 0.20);

  --color-info:            #1a5a9a;
  --color-info-soft:       rgba( 26,  90, 154, 0.09);
  --color-info-border:     rgba( 26,  90, 154, 0.18);

  --color-danger:          #c03030;
  --color-danger-soft:     rgba(192,  48,  48, 0.09);
  --color-danger-border:   rgba(192,  48,  48, 0.18);

  /* ─── FONTES ─── */
  --font-display:   'Cormorant Garamond', Georgia, serif;
  --font-body:      'Outfit', system-ui, sans-serif;
  --font-mono:      'JetBrains Mono', 'Fira Code', monospace;

  /* ─── BORDER RADIUS ─── */
  --radius-xs:   6px;
  --radius-sm:  10px;
  --radius-md:  16px;
  --radius-lg:  22px;
  --radius-xl:  28px;
  --radius-full: 9999px;

  /* ─── ESPAÇAMENTO (escala 4px) ─── */
  --space-1:   4px;
  --space-2:   8px;
  --space-3:  12px;
  --space-4:  16px;
  --space-5:  20px;
  --space-6:  24px;
  --space-8:  32px;
  --space-10: 40px;
  --space-12: 48px;
  --space-16: 64px;

  /* ─── SOMBRAS ─── */
  --shadow-sm:     0 1px 3px  rgba(100, 40, 20, 0.06);
  --shadow-md:     0 4px 12px rgba(100, 40, 20, 0.08), 0 1px 4px rgba(100, 40, 20, 0.05);
  --shadow-glass:  0 4px 24px rgba(100, 40, 20, 0.08), 0 1px 4px rgba(100, 40, 20, 0.06);
  --shadow-button: 0 4px 16px rgba(122,  18, 18, 0.25), inset 0 1px 0 rgba(255, 255, 255, 0.10);
  --shadow-focus:  0 0 0 3px  rgba(196,  32, 32, 0.12);

  /* ─── TRANSIÇÕES ─── */
  --transition-fast:   0.15s ease;
  --transition-normal: 0.20s ease;
  --transition-slow:   0.30s ease;

  /* ─── Z-INDEX ─── */
  --z-base:    1;
  --z-raised:  10;
  --z-overlay: 100;
  --z-modal:   200;
  --z-toast:   300;
  --z-header:  200;

  /* ─── GLASSMORPHISM BACKDROP ─── */
  --backdrop-glass:  blur(20px) saturate(1.2);
  --backdrop-header: blur(24px) saturate(1.4);
}
```

## Guia de uso dos tokens

### Cores — quando usar cada grupo

| Token | Onde usar |
|---|---|
| `--color-bg-base` | Background da página (`<body>`, layout raiz) |
| `--color-bg-surface` | Cards sem glass, modais, dropdowns |
| `--color-glass-1` | Glass cards primários (KPI, content cards) |
| `--color-glass-2` | Glass cards secundários, sidebar |
| `--color-glass-3` | Overlays sutis, nav pills |
| `--color-border` | Bordas padrão de cards e inputs |
| `--color-border-strong` | Bordas em hover, foco, cards ativos |
| `--color-border-divider` | Separadores internos de listas |
| `--color-crimson-*` | Ações primárias, estado ativo, brand |
| `--color-gold` | Acentos editoriais, gradient do título |
| `--color-text-primary` | Corpo do texto, labels importantes |
| `--color-text-secondary` | Subtítulos, texto de suporte |
| `--color-text-muted` | Labels de campo, metadados |
| `--color-text-ghost` | Placeholders, texto desabilitado |

### Regra de hierarquia de texto

```
pageTitle     → font-display, 42px, weight 300, color-text-primary
sectionTitle  → font-display, 22px, weight 300, color-text-secondary
cardTitle     → font-body,    13px, weight 600, uppercase, color-text-muted
bodyText      → font-body,    14px, weight 400, color-text-primary
labelText     → font-body,    11px, weight 500, uppercase, color-text-muted
metaText      → font-body,    11px, weight 300, color-text-ghost
kpiValue      → font-display, 46px, weight 300, color-text-primary
kpiValueMd    → font-display, 26px, weight 300, color-text-primary
```

### Gradient do título (acento em itálico)

```css
.accent {
  font-style: italic;
  background: linear-gradient(135deg, #e8a070 0%, var(--color-gold) 40%, #c06030 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}
```

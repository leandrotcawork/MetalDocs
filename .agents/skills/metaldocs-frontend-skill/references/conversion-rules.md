# Regras de Conversão HTML → React + CSS Modules

Mapeamento direto e completo para converter qualquer HTML no padrão MetalDocs.

---

## 1. Tags HTML → JSX

| HTML | JSX / Mudança necessária |
|---|---|
| `class="..."` | `className={styles.nomeDaClasse}` |
| `for="..."` | `htmlFor="..."` |
| `style="..."` inline | Mover para `.module.css` — só manter se for valor dinâmico |
| `<img src="...">` | `<img src="..." alt="descrição" />` (self-closing + alt) |
| `<input>` | `<input />` (self-closing) |
| `<br>` | `<br />` |
| `onclick="fn()"` | `onClick={fn}` |
| `onchange="fn()"` | `onChange={fn}` |
| `<!-- comentário -->` | `{/* comentário */}` |
| `tabindex` | `tabIndex` |
| `autocomplete` | `autoComplete` |
| `readonly` | `readOnly` |
| Texto direto em loops | Sempre usar `.map()` com `key` único |

---

## 2. CSS: de classes globais para CSS Modules

### Regra geral de conversão

```html
<!-- HTML original -->
<div class="card card--highlighted">
  <h3 class="card__title">Título</h3>
  <span class="badge badge--success">Ativo</span>
</div>
```

```tsx
// JSX convertido
<div className={`${styles.card} ${styles.highlighted}`}>
  <h3 className={styles.title}>Título</h3>
  <span className={`${styles.badge} ${styles.success}`}>Ativo</span>
</div>
```

```css
/* ComponentName.module.css */
.card { ... }
.card.highlighted { ... }   /* variante — não aninhamento BEM, compõe com a classe base */
.title { ... }
.badge { ... }
.badge.success { ... }
```

### Múltiplas classes com lógica condicional

```tsx
// Padrão MetalDocs — sem biblioteca (não usar clsx/classnames)
const cardClass = [
  styles.card,
  isHighlighted ? styles.highlighted : '',
  isLoading ? styles.loading : '',
].filter(Boolean).join(' ')

return <div className={cardClass}>...</div>
```

### Nunca fazer

```tsx
// ❌ Errado — cor hardcoded
<div style={{ background: '#7a1212' }}>

// ❌ Errado — classe global
<div className="card">

// ❌ Errado — style object para layout estático
<div style={{ display: 'flex', gap: '16px' }}>

// ✅ Correto — valor dinâmico (animationDelay é calculado em runtime)
<div className={styles.item} style={{ animationDelay: `${index * 0.07}s` }}>
```

---

## 3. Eventos e interatividade

### onclick simples

```html
<!-- HTML -->
<button onclick="handleSave()">Salvar</button>
```

```tsx
// React — sem () na referência
<button className={styles.button} onClick={handleSave}>Salvar</button>
```

### onclick com argumento

```html
<button onclick="deleteUser('123')">Deletar</button>
```

```tsx
// React — arrow function para passar argumento
<button className={styles.button} onClick={() => onDelete(user.id)}>Deletar</button>
```

### oninput / onchange

```html
<input type="text" oninput="handleInput(this.value)" value="...">
```

```tsx
// React — formulário controlado obrigatório
<input
  type="text"
  className={styles.input}
  value={value}
  onChange={(e) => setValue(e.target.value)}
  placeholder="..."
/>
```

### onsubmit em form

```html
<form onsubmit="handleSubmit(event)">
```

```tsx
// React — nunca usar <form> com onSubmit; usar div + botão onClick
<div className={styles.form}>
  ...
  <button className={styles.submitButton} onClick={handleSubmit}>
    Enviar
  </button>
</div>
```

---

## 4. Listas e repetição

```html
<!-- HTML estático -->
<ul>
  <li class="user-row">João Silva</li>
  <li class="user-row">Maria Costa</li>
</ul>
```

```tsx
// React — sempre com .map() e key
interface User { id: string; name: string }

{users.map((user, index) => (
  <div
    key={user.id}
    className={styles.userRow}
    style={{ animationDelay: `${index * 0.07}s` }}
  >
    {user.name}
  </div>
))}
```

---

## 5. Condicionais

```html
<!-- HTML — elemento que aparece ou não -->
<div class="badge" style="display: none">Ativo</div>
```

```tsx
// React — renderização condicional
{isActive && (
  <span className={styles.badge}>Ativo</span>
)}

// Ou com fallback
{isActive ? (
  <span className={`${styles.badge} ${styles.active}`}>Ativo</span>
) : (
  <span className={`${styles.badge} ${styles.inactive}`}>Inativo</span>
)}
```

---

## 6. Variáveis CSS: substituição de valores literais

Ao converter CSS do HTML, substituir todos os valores hardcoded pelas variáveis do design system:

| Valor literal | Variável MetalDocs |
|---|---|
| `#7a1212`, `#9e1818`, `#c42020` | `var(--crimson)`, `var(--crimson-mid)`, `var(--crimson-bright)` |
| `rgba(156,24,24,0.08)` | `var(--crimson-soft)` |
| `#c9933a` | `var(--gold)` |
| `#faf7f3`, `#fdf9f5`, `#fff` | `var(--bg-base)`, `var(--bg-surface)`, `var(--bg-elevated)` |
| `rgba(255,255,255,0.72)` | `var(--glass-1)` |
| `rgba(255,255,255,0.55)` | `var(--glass-2)` |
| `rgba(150,80,60,0.13)` | `var(--glass-border)` |
| `#1c1008` | `var(--text-primary)` |
| `#3d2010` | `var(--text-secondary)` |
| `#7a5540` | `var(--text-muted)` |
| `#b09080` | `var(--text-ghost)` |
| `#1a7a4a` | `var(--color-success)` |
| `#9a6400` | `var(--color-warning)` |
| `#1a5a9a` | `var(--color-info)` |
| `#c03030` | `var(--color-danger)` |
| `10px`, `14px`, `16px`, etc. | `var(--radius-xs)`, `var(--radius-sm)`, `var(--radius-md)` |
| `backdrop-filter: blur(20px)` | sempre acompanhar com `-webkit-backdrop-filter` |
| `font-family: 'Cormorant...'` | `var(--font-display)` |
| `font-family: 'Outfit...'` | `var(--font-body)` |
| `font-size: 10px`, `11px`, `13px`... | `var(--text-xs)`, `var(--text-sm)`, `var(--text-base)`... |
| `font-weight: 300/400/500/600` | `var(--weight-light/regular/medium/semibold)` |
| `letter-spacing: 0.10em` | `var(--tracking-label)` |
| `z-index: 200` | `var(--z-header)` |
| `150ms`, `200ms`, `300ms` | `var(--duration-fast/normal/slow)` |

---

## 7. Animações

### keyframes globais disponíveis (em `globals.css`)

```css
/* Não redefinir nos .module.css — já existem globalmente */
@keyframes fadeInUp { ... }    /* entrada de baixo para cima */
@keyframes fadeInLeft { ... }  /* entrada da esquerda */
@keyframes slideDown { ... }   /* header desliza para baixo */
@keyframes pipping { ... }     /* pulse para status dots */
@keyframes float { ... }       /* flutuação lenta para orbs */
@keyframes shimmer { ... }     /* loading skeleton */
```

### Como usar no module.css

```css
/* Referência direta — funciona porque são globais */
.card {
  animation: fadeInUp var(--duration-enter) var(--ease-out) both;
}

.skeleton {
  animation: shimmer 1.5s infinite;
}
```

---

## 8. Estrutura de arquivo final por tipo

### Page

```
pages/
└── nomePage/
    ├── NomePage.tsx         ← orquestra, busca dados, sem estilo
    ├── NomePage.module.css  ← só layout geral da page (grid, padding)
    └── sections/
        ├── NomeSection.tsx
        └── NomeSection.module.css
```

### Componente reutilizável

```
components/
└── ui/
    └── NomeComponente/
        ├── NomeComponente.tsx
        ├── NomeComponente.module.css
        └── index.ts          ← barrel: export { default } from './NomeComponente'
```

### Hook

```
hooks/
└── useNomeHook.ts   ← sem pasta, sem module.css
```

---

## 9. Checklist de conversão — item a item

Antes de entregar o código convertido, confirme cada item:

**Estrutura**
- [ ] Arquivo `.tsx` e `.module.css` na mesma pasta
- [ ] `index.ts` com barrel export se for componente novo
- [ ] Nenhuma lógica de negócio no componente de UI
- [ ] Props com `interface Props` no topo do arquivo

**JSX**
- [ ] Todos os `class` → `className={styles.xxx}`
- [ ] Todos os `for` → `htmlFor`
- [ ] Todos os `onclick/onchange` → `onClick/onChange` camelCase
- [ ] Nenhum `<form>` com `onSubmit` — usar div + botão
- [ ] Self-closing tags com `/>`
- [ ] Listas com `.map()` e `key` único
- [ ] Condicionais com `&&` ou ternário, nunca `display: none`

**CSS Module**
- [ ] Nenhuma cor hexadecimal literal — só variáveis
- [ ] Nenhum tamanho de fonte literal — só variáveis
- [ ] Nenhum `z-index` literal — só variáveis
- [ ] `backdrop-filter` sempre acompanhado de `-webkit-backdrop-filter`
- [ ] Variantes como `.root.highlighted`, nunca `.root--highlighted`
- [ ] Animações referenciando os keyframes globais

**Qualidade**
- [ ] Estado loading com skeleton
- [ ] Estado empty com mensagem
- [ ] Estado error com botão retry (quando dado vem de API)
- [ ] Formulários controlados (`value` + `onChange`)
- [ ] `aria-label` em botões sem texto visível
- [ ] `htmlFor` + `id` correspondente em todos os labels/inputs

---
name: metaldocs-frontend-skills
description: Converte HTML para o padrão React + CSS do projeto MetalDocs. Use este skill SEMPRE que o usuário pedir para converter um HTML em componente React, criar um novo componente, page ou layout para o MetalDocs, reescrever markup em estilo MetalDocs, ou quando mencionar "componente", "página", "layout", "converter para React", "estilo MetalDocs", "tsx", ou qualquer pedido de código frontend no contexto do projeto MetalDocs. Também use quando o usuário quiser criar algo novo que siga o design system do MetalDocs — glassmorphism, fundo branco/creme com orbs animados, tipografia Cormorant Garamond + Outfit, paleta crimson/gold.
---

# MetalDocs Frontend — HTML → React + CSS

Skill de conversão e criação de componentes React para o projeto MetalDocs (`frontend/apps/web`).
Workflow, plano, commits e registro em `tasks/todo.md` seguem `$md`; este skill cuida da conversão estrutural/visual.

## Quando usar

- Converter HTML em React + CSS Modules
- Criar componente, section, card ou layout seguindo o visual MetalDocs
- Reescrever markup para o padrão do projeto

Se o pedido for uma página/feature completa ou um fluxo grande, use `$md` como orquestrador e este skill como executor de frontend. Se a tarefa merecer `/plan`, avise o usuário.

## Antes de começar — leia os references

| Arquivo | Quando carregar |
|---|---|
| `references/design-tokens.md` | **Sempre** — variáveis CSS, tipografia, cores, sombras |
| `references/component-patterns.md` | **Sempre** — estrutura de arquivos, convenções TypeScript |
| `references/glass-patterns.md` | Cards, backgrounds, glassmorphism, animações de orb |
| `references/conversion-rules.md` | Mapeamento direto de tags/classes HTML → JSX + CSS Module |

---

## Fluxo de trabalho (nesta ordem)

### 1. Análise do HTML recebido

Antes de gerar qualquer código, identifique:
- **Blocos estruturais**: header, sidebar, cards, listas, forms, modais
- **Estado dinâmico**: o que muda (contadores, toggles, dados de API, inputs)
- **Eventos**: clicks, submits, onChange com lógica
- **Repetição**: blocos que aparecem N vezes → componente próprio

### 2. Decomposição em componentes

Hierarquia MetalDocs:

```
Page (*Page.tsx)
  └─ Section (*Section.tsx)        ← bloco visual com CSS próprio
       ├─ Card / Item (*Card.tsx)  ← unidade repetível
       └─ UI Atoms (Button, Badge, Avatar, StatusPip, ...)
```

- **Page**: orquestra dados + layout. Sem estilo próprio.
- **Section**: bloco independente, tem `.module.css` próprio.
- **Card/Item**: unidade que se repete. Sempre componentizada.
- **Atoms**: sem estado, só props. Reutilizáveis em toda a app.

> Regra: se um bloco aparece mais de uma vez OU pode aparecer em outro contexto → componente separado.

### 3. Gerar os arquivos

Para cada componente, gere **sempre dois arquivos**:
1. `ComponentName.tsx`
2. `ComponentName.module.css`

Padrões detalhados estão em `references/component-patterns.md`.
Variáveis e tokens estão em `references/design-tokens.md`.

### 4. Checklist antes de entregar

- [ ] Nenhum `style={{}}` inline (exceto valores realmente dinâmicos)
- [ ] Todas as props tipadas com `interface Props`
- [ ] CSS usa **apenas** variáveis do design system
- [ ] Nenhuma cor hardcoded nos arquivos `.module.css`
- [ ] Listas com animação em stagger (`animationDelay`)
- [ ] Sempre: estado loading / empty / error em dados externos
- [ ] Formulários controlados (`value` + `onChange`)
- [ ] `aria-label`, `htmlFor` e `role` onde necessário

---

## Regras invioláveis

- **Nunca** Tailwind, styled-components ou CSS-in-JS
- **Nunca** CSS global importado em componente — sempre CSS Module
- **Nunca** prop `className` exposta para override externo
- **Nunca** cor ou tamanho hardcoded no `.module.css`
- **Sempre** `default export` no componente principal
- **Sempre** `interface Props` (não `type`) para props de componente
- **Sempre** prefixar callbacks com `on`: `onSave`, `onDelete`, `onToggle`
- **Sempre** separar lógica de UI: hooks em arquivo `use*.ts` separado se > 20 linhas de lógica
- **Sempre** editar o repositório real, não responder com “template solto”, salvo se o usuário pedir apenas um exemplo

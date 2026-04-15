# Design System: The Editorial Architect

## 1. Overview & Creative North Star
**Creative North Star: "The Intellectual Atelier"**

This design system is not a generic utility; it is a curated workspace designed for deep focus and high-output creation. While the request calls for "Notion-meets-Word," we are elevating that concept beyond the standard "SaaS-blue" grid. 

We break the "template" look by treating the UI as an editorial layout rather than a software dashboard. By leveraging the juxtaposition of technical precision (Inter) and literary heritage (Newsreader), we create a "High-End Editorial" experience. We prioritize **Tonal Layering** over structural lines and **Intentional Asymmetry** over rigid grids to ensure the interface feels organic, premium, and custom-tailored to the professional writer.

---

## 2. Colors & Surface Philosophy

The palette is anchored in a sophisticated Maroon (`#800000`), balanced by an expansive range of architectural greys and "Paper" whites.

### The "No-Line" Rule
**Explicit Instruction:** Do not use 1px solid borders for sectioning or layout containment. 
Boundaries are defined exclusively through background shifts. A sidebar (`surface-container-low`) sits against the main editor (`surface-container-lowest`) without a stroke. This creates a seamless, modern flow that feels like a physical object rather than a digital wireframe.

### Surface Hierarchy & Nesting
Treat the UI as a series of physical layers. Use the following tokens to create "nested" depth:
*   **Editor Surface:** `surface-container-lowest` (#FFFFFF) – The purest focal point.
*   **App Background:** `surface` (#F8F9FF) – The foundation.
*   **Side Panels/Navigation:** `surface-container-low` (#EEF4FF) – Secondary focus.
*   **Modals/Floating Menus:** `surface-bright` (#F8F9FF) – Elements that command attention.

### The "Glass & Gradient" Rule
To avoid a flat, "out-of-the-box" feel, floating elements (like hover toolbars or property inspectors) should use semi-transparent `surface` colors with a `backdrop-filter: blur(12px)`. 
*   **Signature Textures:** For primary actions, use a subtle linear gradient from `primary` (#570000) to `primary-container` (#800000). This provides a "jewel-tone" depth that feels high-end and authoritative.

---

## 3. Typography: The Dual-Tone System

The typographic system is a conversation between the "Tool" (UI) and the "Thought" (Content).

*   **The Tool (Inter):** Used for all navigation, labels, and metadata. It is precise, legible at small scales, and utilitarian.
*   **The Thought (Newsreader):** Used for Headlines and Display. It introduces an academic, sophisticated air that encourages long-form reading and professional publishing.

| Role | Token | Font | Size | Intent |
| :--- | :--- | :--- | :--- | :--- |
| **Hero Title** | `display-lg` | Newsreader | 3.5rem | Chapter starts, major titles. |
| **Section Header** | `headline-sm` | Newsreader | 1.5rem | Document sub-headers. |
| **UI Primary** | `title-md` | Inter | 1.125rem | Sidebar categories, menu titles. |
| **Document Body**| `body-lg` | Inter* | 1rem | High-readability document text. |
| **Meta/Labels** | `label-sm` | Inter | 0.6875rem | "Word count," "Last edited," tooltips. |

*\*Note: For the document body, use `body-lg` with a line-height of 1.6 to ensure editorial comfort.*

---

## 4. Elevation & Depth: Tonal Layering

Traditional drop shadows are often too "loud" for a minimal workspace. We use **Tonal Layering** to convey hierarchy.

*   **The Layering Principle:** Instead of a shadow, place a `surface-container-lowest` card on top of a `surface-container-low` background. The shift in "whiteness" provides a soft, natural lift.
*   **Ambient Shadows:** When a floating element is required (e.g., a right-click context menu), use a shadow with a 24px blur and 4% opacity. The shadow color must be a tinted version of `on-surface` (#151C25), never pure black.
*   **The "Ghost Border" Fallback:** If an edge is absolutely necessary (e.g., for accessibility in high-contrast modes), use the `outline-variant` token at 15% opacity. **Prohibit 100% opaque borders.**
*   **Glassmorphism:** Navigation bars should use a 70% opacity `surface` color with a heavy blur. This allows document content to "ghost" beneath the UI as the user scrolls, integrating the layout.

---

## 5. Components

### Buttons
*   **Primary:** Gradient from `primary` to `primary-container`. Text: `on-primary`. Radius: `md` (0.75rem).
*   **Secondary:** Ghost style. No background, Maroon (`primary`) text. Use `surface-container-high` on hover.
*   **Tertiary:** `surface-container-highest` background with `on-surface` text for low-priority actions.

### Input Fields & Editor Blocks
*   **Inputs:** No border. Use `surface-container-low` background. On focus, transition the background to `surface-container-lowest` and add a "Ghost Border" using the `primary` accent at 20% opacity.
*   **Text Areas:** In the editor, there are no containers. Content is "naked" on the `surface-container-lowest` white background.

### Cards & Lists (The Editorial Feed)
*   **No Dividers:** Forbid horizontal lines between list items. Use vertical spacing (12px–16px) or a subtle background shift (`surface-container-low`) on hover to define the hit area.
*   **Selection:** Use a `primary-fixed` (#FFDAD4) background with an `on-primary-fixed-variant` (#8F0F07) left-side "accent bar" (4px width).

### Floating Toolbars
*   Contextual formatting menus must use the Glassmorphism rule. Background: `surface` at 80% + 12px blur. Roundedness: `lg` (1rem).

---

## 6. Do’s and Don’ts

### Do
*   **Do** use asymmetrical margins. Larger left-hand gutters for document content create a "notebook" feel.
*   **Do** use Maroon sparingly. It is a "power" color—use it for the one thing you want the user to do next.
*   **Do** lean into `surface-container` shifts to group related items instead of drawing boxes around them.

### Don't
*   **Don't** use pure black (#000000) for text. Use `on-surface` (#151C25) to maintain the high-end, softened aesthetic.
*   **Don't** use standard "Word" blue for links. Links should be Maroon (`primary`) with a 1px underline.
*   **Don't** use sharp corners. Every interaction point must use at least the `md` (0.75rem) corner radius to maintain the "Soft Minimalism" vibe.
*   **Don't** clutter the top-level navigation. If it isn't essential for writing, hide it in a `surface-container-low` sidebar.
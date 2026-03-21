# CSS Tokens — MetalDocs

## Colors (from styles.css)
--vinho          #6b1f2a   primary brand
--accent         #c8364a   interactive / CTA
--bg             #f0ebeb   page background
--surface        #ffffff   card / panel background
--surface-2      #faf6f6   subtle background
--border         #e8dede   standard border
--border-2       #d8caca   stronger border
--text           #1a0e0e   primary text
--text-soft      #483030   secondary text
--muted          #7a6060   muted text / labels
--warning        #c89020   warning
--danger         #c8364a   error / danger
--success        #1a6b35   success

## Spacing (use multiples of 4)
Use: 4px 8px 12px 16px 20px 24px 32px 48px
Always via: padding: 16px → padding: 1rem (or calc from 4px base)

## Typography
Font family: "DM Sans" (body), "DM Mono" (code/mono)
Sizes: 10px 11px 12px 13px 14px 16px 19px 24px+
Weights: 400 (regular) 500 (medium) 600 (semibold) 700 (bold)

## Component tokens to define per feature
When writing a new feature module, define its tokens at the top of its .module.css:
```css
/* DocumentsWorkspace.module.css */
.shell {
  --row-height: 46px;
  --col-doc: minmax(0, 1fr);
  --col-type: 120px;
  display: grid;
}
```

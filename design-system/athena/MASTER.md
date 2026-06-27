# ATHENA Design System

> Source of truth for `ui/src/app`. Page-specific files under
> `design-system/athena/pages/` override this document only where they state an
> explicit exception.

## Product direction

- Product: internal blockchain and prediction-market operations console.
- Audience: operators working on desktop workstations at 1280px and wider.
- Style: professional, dark-first, data-dense, calm, and highly scannable.
- Stack: React 18 and Ant Design 6. Keep Ant Design components and icons.
- Themes: dark is the default; light remains a fully supported secondary theme.
- Avoid ornamental glass effects, cyberpunk treatments, neon glow, display
  fonts, excessive whitespace, and layout-shifting hover effects.

## Foundations

### Color tokens

| Role | Dark | Light |
|---|---|---|
| Canvas | `#0B0F14` | `#F4F6F8` |
| Surface | `#111820` | `#FFFFFF` |
| Elevated surface | `#16202A` | `#F8FAFC` |
| Soft surface | `#1B2632` | `#EDF1F5` |
| Border | `#263341` | `#D8E0E8` |
| Strong border | `#354556` | `#C2CDD8` |
| Primary text | `#EEF3F8` | `#18212B` |
| Secondary text | `#94A3B8` | `#637083` |
| Brand / primary action | `#E76F51` | `#D85D42` |
| Brand hover | `#EF8165` | `#C94F36` |
| Information | `#3B82F6` | `#2563EB` |
| Success | `#22C55E` | `#168A45` |
| Warning | `#F59E0B` | `#B86A00` |
| Error | `#EF4444` | `#C9363E` |
| Focus ring | `#60A5FA` | `#2563EB` |

Color is never the only status indicator. Pair it with text, an icon, or a
shape. Primary text must meet 4.5:1 contrast; non-text controls and large text
must meet 3:1.

### Typography

- Interface: `Inter, Heebo, ui-sans-serif, system-ui, -apple-system,
  BlinkMacSystemFont, "Segoe UI", sans-serif`.
- Data: `ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
  "Liberation Mono", monospace`.
- Use the data stack only for addresses, hashes, identifiers, prices, balances,
  timestamps, and chart values.
- Base size is 14px. Page titles are 22px, section titles 14–16px, table
  headings 11–12px, and supporting labels 12px.
- Prefer weight and spacing over uppercase. Uppercase is reserved for compact
  navigation group labels and table headers.

### Spacing and geometry

- Spacing scale: 4, 8, 12, 16, 20, 24, and 32px.
- Sidebar: 248px expanded, 72px collapsed.
- Global header: 56px.
- Standard control: 36px; compact table row target: 40px.
- Radius: 8px for controls and panels, 6px for tags and small status blocks.
- Page gutter: 20px at 1280px, 24px at 1440px and wider.
- Shadows are reserved for overlays and the active navigation boundary. Normal
  panels rely on surface and border contrast.

### Motion

- Use 150–200ms transitions for color, opacity, border, and shadow.
- Do not animate width or height during routine interaction.
- Hover and pressed states must not move surrounding layout.
- Disable non-essential motion under `prefers-reduced-motion: reduce`.

## Application shell

### Navigation

Navigation is grouped without changing route or permission semantics:

1. `MARKETS`: Polymarket and FIFA.
2. `TOKEN & RISK`: Token and Wallets.
3. `OPERATIONS`: Notifications and Service Status.
4. `SYSTEM`: Settings, User Info, and Help.

The active item uses a brand-colored left indicator and a restrained surface
highlight. Collapsed mode keeps icons and accessible labels. Navigation is a
semantic landmark and is bypassable through a skip link.

### Header

- Left: sidebar toggle and route breadcrumb.
- Right: theme control and user menu.
- Icon-only controls require a tooltip and an accessible name.
- The user menu links to User Info, Settings, and Help. Logout remains on the
  existing User Info page.

### Page command area

- One page title, optional subtitle/status, secondary actions, then one primary
  action.
- Refresh is a labelled tooltip action when icon-only.
- Filters sit in a separate compact toolbar directly above the content.
- Loading, empty, warning, and error states appear inside the content region
  they affect.

## Component rules

### Tables

- Use compact density, sticky headers, tabular numerals, visible row hover, and
  a clear selected state.
- Wide data remains horizontally scrollable inside its panel; the application
  canvas must not grow unexpectedly.
- Clickable rows support keyboard activation or expose a real link/button for
  the action.
- Pagination belongs to the table footer and uses the same surface boundary.

### Panels and cards

- Default panels use the surface color, a 1px border, and no decorative blur.
- Nested panels use the elevated or soft surface, not an additional shadow.
- Live and monitoring cards emphasize current status, timestamp, and primary
  value before supporting metadata.
- Charts use the information, warning, error, and success colors with labels so
  series remain distinguishable without color alone.

### Forms and overlays

- Inputs have persistent labels, distinct backgrounds, visible focus rings,
  inline validation, and explicit required markers.
- Submit actions show loading and success/error feedback.
- Destructive actions are spatially separated and use confirmation where the
  existing workflow already provides it.
- Drawers and modals use an opaque elevated surface and a 50–60% black scrim.
  Focus must enter the overlay and return to the trigger.

### Feedback and accessibility

- Maintain a logical heading hierarchy and DOM order.
- Add `aria-label` to every icon-only action.
- Do not remove native focus outlines without an equal or stronger replacement.
- Async operations longer than 300ms show a spinner, skeleton, or inline status.
- Status updates that change without navigation use an appropriate live region
  when they carry operational meaning.

## Page families

- Data tables: compact command area, optional filters, one primary data panel.
- Live monitoring: operational status first, then dense cards and charts.
- Detail/forms: summary first, grouped details second, actions last.
- Login: centered, compact authentication panel using the same dark surfaces
  and brand identity; no marketing-style decoration.

## Delivery checklist

- Ant Design remains the only component and structural icon system.
- Dark and light themes use semantic tokens with equivalent state contrast.
- Navigation works expanded and collapsed at 1280, 1440, and 1920px.
- Tables, drawers, modals, forms, charts, and async states follow the component
  rules above.
- Focus is visible, icon actions are named, and reduced motion is respected.
- No mobile reflow is required; the supported minimum canvas is 1280px.

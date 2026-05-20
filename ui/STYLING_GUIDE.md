# Talos Operator UI Styling Guide

The UI uses **Tailwind CSS** with a small custom theme. There is no Bootstrap and
no `.talos-*` utility layer; earlier versions of this guide described a
since-removed `TalosUI.css`. All styling lives in component classes plus a few
global rules in `src/index.css`.

## Stack

- Tailwind 3 (`tailwind.config.js`, processed by PostCSS / `react-scripts`)
- Inter for sans, JetBrains Mono for code (see `fontFamily` in the Tailwind config)
- `lucide-react` for icons (sized 12–16 in chrome, 20+ in empty states)

## Color tokens

Defined under `theme.extend.colors` in `tailwind.config.js`:

| Token         | Value                          | Use                              |
| ------------- | ------------------------------ | -------------------------------- |
| `brand`       | `#FF6B35`                      | Primary accent, active nav, CTA  |
| `brand-hover` | `#ff7c4d`                      | Hover state for `brand`          |
| `brand-dim`   | `rgba(255, 107, 53, 0.12)`     | Tinted backgrounds, soft chips   |

Everything else uses Tailwind's stock `zinc` palette for surfaces and text, and
the stock `green` / `red` / `yellow` / `sky` / `purple` / `emerald` scales for
status and kind badges.

### Surface scale (dark theme)

| Layer            | Class                                |
| ---------------- | ------------------------------------ |
| Page background  | `bg-zinc-950`                        |
| Panel / card     | `bg-zinc-900` + `border-zinc-800`    |
| Row hover        | `bg-zinc-800/30` or `bg-zinc-800/50` |
| Strong border    | `border-zinc-700`                    |
| Body text        | `text-zinc-100` / `text-zinc-200`    |
| Secondary text   | `text-zinc-400` / `text-zinc-500`    |
| Muted / disabled | `text-zinc-600` / `text-zinc-700`    |

### Status tones

`ResourceList.getStatus` maps Kubernetes condition state to one of four tones:

| Tone     | Text class        | Dot class      | Meaning                                     |
| -------- | ----------------- | -------------- | ------------------------------------------- |
| `green`  | `text-green-400`  | `bg-green-500` | `Ready=True`                                |
| `yellow` | `text-yellow-400` | `bg-yellow-500`| `Ready=Unknown` or in-progress reason       |
| `red`    | `text-red-400`    | `bg-red-500`   | `Ready=False`, or any failing condition     |
| `gray`   | `text-zinc-500`   | `bg-zinc-600`  | No conditions yet                           |

Reuse the same palette anywhere status is shown so the visualizer, lists, and
detail pages stay consistent.

## Layout patterns

- **App shell** (`App.tsx`): `flex h-screen` with a fixed `w-56` sidebar and a
  flex‑1 column containing a thin `h-11` header bar and a scrollable `<main>`.
- **Cards**: `bg-zinc-900 border border-zinc-800 rounded-xl` with internal
  padding `p-5` (compact) or `p-6` (page-level).
- **Tables**: zebra-less, `divide-y divide-zinc-800` for row separators, header
  row `text-xs uppercase tracking-wider text-zinc-500`.
- **Forms**: labels are `text-xs font-medium text-zinc-400`, inputs use the
  `inputCls` / `selectCls` / `textareaCls` constants in `TalosResourceForm.tsx`
  — copy them when adding new form components instead of redefining the look.

## Conventions

1. Prefer Tailwind utilities directly in JSX; avoid component-level CSS files.
2. New global rules go in `src/index.css` only when they cannot be expressed as
   utilities (scrollbar styling, ReactFlow overrides, etc.).
3. Use the tokens above — don't hardcode `#FF6B35` or other hex values in
   components. If a new accent is needed, add it to `tailwind.config.js`.
4. Icon sizes: 12 for inline chrome buttons, 14–16 for nav and headers, 20+
   for empty-state visuals.
5. Keep transitions short: `transition-colors` is enough for most hover states.

## File map

```text
ui/src/
├── index.css          # Tailwind directives, body defaults, scrollbar, ReactFlow overrides
├── App.css            # currently empty — kept for CRA compatibility
├── App.tsx            # shell, toast plumbing, routing
└── components/
    ├── Sidebar.tsx
    ├── Overview.tsx
    ├── ResourceList.tsx
    ├── TalosResourceForm.tsx
    └── ClusterVisualizer.tsx
```

## Dev workflow

```bash
cd ui
npm start       # dev server (proxies /api to localhost:8080)
npm run build   # production build
```

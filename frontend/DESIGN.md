# Design: "Clearing House"

The platform holds other people's money in escrow, so the interface should
feel like a calm, exact financial tool: paper-white (or night-blue) surfaces,
one ink colour, one blue, and numbers that line up.

## Signature

**The term sheet** (`src/components/TermSheet.tsx`) is the one picture the
user learns once and sees everywhere (Home, the calculator, the swap page):

1. a split bar: the share of party A and party B if the project succeeds;
2. a "vault" of three cells: guarantee A | the escrow | guarantee B.

Amber always means *money that is held*. Nothing else uses amber.

## Tokens

All tokens live at the top of `src/index.css`. The light theme is on
`:root[data-theme='light']`, the dark theme on `:root[data-theme='dark']`.
Rules never use a colour directly, only a variable.

| Token | Light | Dark | Use |
|---|---|---|---|
| `--bg` | `#f3f5f8` | `#0b1320` | page background |
| `--surface` | `#ffffff` | `#111b2b` | cards, navbar, inputs |
| `--surface-2` | `#f7f9fb` | `#152134` | hover rows |
| `--surface-soft` | `#eef1f5` | `#1b2739` | bar tracks, quiet tags |
| `--border` / `--border-strong` | `#dce2ea` / `#c5ceda` | `#243248` / `#33445e` | lines |
| `--text` / `--text2` / `--text3` | `#0e1a2b` / `#5a6779` / `#8792a2` | `#e6ebf2` / `#97a3b6` / `#6b788c` | ink, labels, hints |
| `--accent` | `#2147c7` | `#7d9bff` | links, focus, party A, ML model |
| `--party-a` / `--party-b` | `#2147c7` / `#9db0ee` | `#7d9bff` / `#4f68c7` | split bar, charts |
| `--escrow` / `--escrow-soft` | `#a86a12` / `#fbf1df` | `#e3a341` / amber 16% | held money only |
| `--success` / `--danger` / `--warning` | `#13795b` / `#b42318` / `#9a5b06` | `#3cc29a` / `#f97066` / `#f0b04c` | meaning, each with a `-soft` background |
| `--primary` | ink | light ink | the one main button on a screen |

Old names (`--bg2`, `--bg3`, `--accent2`) point to the new ones, so older
code keeps working.

## Type

IBM Plex Sans (400, 500, 600) from `@fontsource/ibm-plex-sans`, with
`font-variant-numeric: tabular-nums` on the whole page so money and
percentages line up in columns.

| Role | Size / weight |
|---|---|
| Home headline | 46px / 600 (34px on phones) |
| Page title | 28px / 600 |
| Big number | 32px / 600 |
| Card title | 15–16px / 600 |
| Body | 15px / 400 |
| Labels | 13px / 400–500 |

## Space and shape

A 4px grid: 4, 8, 12, 16, 20, 24, 32, 48, 56. Cards have 24px padding and
20px gaps. Radius: 6px (controls), 8px (cards), 12px (large blocks).
Shadows are almost invisible; borders do the work.

## Motion

- Short and ease-out: `--ease-out: cubic-bezier(0.23, 1, 0.32, 1)`,
  120ms for presses, 200ms for hovers, 300–400ms for entrances.
- Entrances rise 6–10px and fade. Nothing starts from `scale(0)` and
  nothing bounces.
- Buttons answer a press with `scale(0.97)`.
- The split bar opens once from left to right; probability bars grow once.
- `prefers-reduced-motion`: CSS turns transitions off, and
  `<MotionConfig reducedMotion="user">` does the same for framer-motion.

## Theme switch

`index.html` sets `data-theme` before the page is drawn (saved choice,
otherwise the system setting), so there is no flash. The button in the
navbar calls `toggleTheme()` from `ThemeProvider`, which saves the choice
in `localStorage` (`swap-theme`). Charts read their colours through
`useChartTheme()`, so they redraw with the new colours at once.
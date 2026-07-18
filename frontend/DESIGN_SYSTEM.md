# SimpleTranslate Design System

SimpleTranslate is a high-density desktop translation workspace. Neutral surfaces
carry the interface; indigo identifies the product, focus, selection, and the
primary translation action. Semantic states always use their own colors.

## Principles

- Text first: source and translated content receive most of the window.
- Dense, not cramped: controls use stable compact dimensions and never resize
  when labels, loading states, or results change.
- One predictable workspace: navigation, language routing, engine selection,
  modes, editors, and status always stay in the same order.
- State is explicit: credentials, processing, partial engine failure, copied
  text, saved settings, and empty data have distinct feedback.
- Keyboard complete: all controls have visible focus, overlays trap focus,
  Escape closes them, and focus returns to the trigger.

## Foundation

| Group | Tokens | Rule |
| --- | --- | --- |
| Accent | `--primary`, `--primary-hover`, `--primary-soft` | Brand, focus, selection, and primary actions only |
| Semantic | `--success`, `--warning`, `--danger`, `--info` | Success, attention, failure, and processing respectively |
| Surfaces | `--bg-rail`, `--bg-workspace`, `--bg-panel`, `--bg-surface`, `--bg-elevated` | Build hierarchy with neutral contrast, never gradients |
| Text | `--text-main`, `--text-sec`, `--text-muted` | Content, controls, and metadata |
| Radius | `--radius-xs` through `--radius-lg` | No panel or dialog exceeds 8px |
| Spacing | `--sp-1` through `--sp-10` | Four-pixel base grid |
| Motion | `--t-fast`, `--t-base`, `--t-slow` | 120-240ms; reduced motion disables nonessential animation |

The default theme is dark. `.light-mode` overrides the same semantic tokens.
Components must not introduce theme-specific color branches.

## Layout

- Utility rail: fixed 56px with product mark, history, theme, and settings.
- Command bar: 52px on desktop; language route, engine, independent modes, and
  translation action remain in one row.
- Workspace: two full-bleed equal editor tracks separated by a 1px divider.
- Editor headers are 42px and footers are 38px. The status bar is 26px.
- At 720px and below, the command bar uses two rows and editors stack. The rail
  remains fixed so navigation never changes location.
- At 460px and below, secondary labels collapse before controls. Icon buttons
  keep accessible names and tooltips.

## Components

- Icon-only commands use Lucide icons, stable square dimensions, accessible
  names, and hover/focus tooltips.
- Engine choice is a segmented control. Auto translation, clipboard watching,
  and comparison are independent toggle buttons.
- Editors and page sections are unframed. Cards are reserved for repeated
  history and engine-result items.
- Settings use unframed sections separated by dividers. Tests operate on the
  draft; only Save persists it.
- History remains a right drawer. Settings remain a centered modal. Both trap
  focus and return it when closed.
- All text must fit at 1440x900, 1024x768, 720x800, and 390x844 without horizontal
  page overflow or incoherent overlap.

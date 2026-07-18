# SimpleTranslate Design System

The Aurora interface is a quiet desktop workspace. Neutral surfaces carry the
layout; indigo is reserved for identity, focus, selection, and primary actions.
The source of truth is `src/style.css`.

## Principles

- Dense and predictable: translation controls stay in fixed locations and the
  workspace prioritizes text over decoration.
- Token driven: components consume global colors, spacing, type, radii, motion,
  and z-index values instead of declaring competing theme values.
- State is visible: loading, updating, partial failure, missing credentials,
  saved settings, and copied text each have explicit feedback.
- Keyboard complete: every action has a focus state; dialogs trap focus, close
  with Escape, and return focus to their trigger.
- Responsive by constraint: editor tracks and tool groups have stable minimums;
  the workspace stacks at 720px without horizontal overflow.

## Core Tokens

| Group | Tokens | Rule |
| --- | --- | --- |
| Accent | `--primary`, `--primary-hover`, `--primary-soft` | Use only for brand, selection, focus, and primary commands |
| Semantic | `--success`, `--warning`, `--danger`, `--info` | Never reuse accent color for semantic state |
| Surfaces | `--bg-base`, `--bg-sidebar`, `--bg-panel`, `--bg-surface`, `--bg-elevated` | Establish hierarchy with contrast, not gradients |
| Text | `--text-main`, `--text-sec`, `--text-muted` | Main copy, supporting copy, metadata |
| Radius | `--radius-xs` through `--radius-lg` | Panels and dialogs never exceed 8px; pills use `--radius-full` |
| Spacing | `--sp-1` through `--sp-10` | Four-pixel base grid |
| Motion | `--t-fast`, `--t-base`, `--t-slow` | 120-260ms; reduced-motion disables nonessential animation |

## Layout

- Desktop: 228px sidebar, compact header, two equal editor tracks.
- Compact: labels collapse before controls; text tracks remain side by side.
- Narrow (`<=720px`): 60px icon rail, two-row header, stacked source/result
  tracks with a minimum 230px working height.
- Status bar remains available at all widths and shows the active language
  route and translation engine without competing with the editor content.

## Components

- Primary buttons use a solid accent fill. Broad gradients and ambient glow
  backgrounds are not part of Aurora.
- Repeated history and comparison results may use bordered item surfaces;
  page sections remain unframed.
- Icon-only controls use Lucide icons, accessible names, and tooltips.
- Settings always edit a draft. Connection tests use that draft but never
  persist it; only the Save command updates application configuration.
- Dark mode is the default. `.light-mode` overrides the same semantic tokens,
  so component CSS must not branch on theme.

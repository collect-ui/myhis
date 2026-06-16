---
name: Technical IDE Light System
colors:
  surface: '#fbf9f9'
  surface-dim: '#dbdad9'
  surface-bright: '#fbf9f9'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f5f3f3'
  surface-container: '#efeded'
  surface-container-high: '#e9e8e7'
  surface-container-highest: '#e3e2e2'
  on-surface: '#1b1c1c'
  on-surface-variant: '#434654'
  inverse-surface: '#303031'
  inverse-on-surface: '#f2f0f0'
  outline: '#737685'
  outline-variant: '#c3c6d6'
  surface-tint: '#0c56d0'
  primary: '#003d9b'
  on-primary: '#ffffff'
  primary-container: '#0052cc'
  on-primary-container: '#c4d2ff'
  inverse-primary: '#b2c5ff'
  secondary: '#585e70'
  on-secondary: '#ffffff'
  secondary-container: '#dce2f8'
  on-secondary-container: '#5e6477'
  tertiary: '#7b2600'
  on-tertiary: '#ffffff'
  tertiary-container: '#a33500'
  on-tertiary-container: '#ffc6b2'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#dae2ff'
  primary-fixed-dim: '#b2c5ff'
  on-primary-fixed: '#001848'
  on-primary-fixed-variant: '#0040a2'
  secondary-fixed: '#dce2f8'
  secondary-fixed-dim: '#c0c6db'
  on-secondary-fixed: '#151b2b'
  on-secondary-fixed-variant: '#404658'
  tertiary-fixed: '#ffdbcf'
  tertiary-fixed-dim: '#ffb59b'
  on-tertiary-fixed: '#380d00'
  on-tertiary-fixed-variant: '#812800'
  background: '#fbf9f9'
  on-background: '#1b1c1c'
  surface-variant: '#e3e2e2'
typography:
  headline-lg:
    fontFamily: Manrope
    fontSize: 24px
    fontWeight: '700'
    lineHeight: 32px
    letterSpacing: -0.02em
  headline-md:
    fontFamily: Manrope
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
  body-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  body-sm:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 18px
  code-md:
    fontFamily: JetBrains Mono
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 20px
  label-bold:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
  label-sm:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 14px
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  unit: 4px
  space-xs: 4px
  space-sm: 8px
  space-md: 16px
  space-lg: 24px
  space-xl: 32px
  edge-margin: 12px
  panel-gutter: 1px
---

## Brand & Style

This design system is engineered for high-productivity technical environments where clarity, precision, and long-term visual comfort are paramount. The aesthetic follows a **Corporate Modern** approach with a lean toward **Minimalism**, stripping away non-essential decorative elements to prioritize the developer's code. 

The system targets professional software engineers and data scientists working in high-glare daylight environments. It evokes a sense of reliability and architectural stability through a structured grid, generous whitespace, and a high-contrast color palette. The UI is designed to feel invisible, acting as a functional frame for the content it contains.

## Colors

The color palette is anchored by a "Daylight High-Contrast" logic. 
- **Core Surfaces:** Pure white (#FFFFFF) is reserved for the primary editor and document areas to maximize contrast. #FAFAFA and #F5F5F5 are used for sidebars, toolbars, and panels to create a clear structural hierarchy without using heavy lines.
- **Accents:** Corporate Blue (#0052CC) serves as the singular primary action color, used for active states, focus rings, and primary buttons.
- **Semantic Colors:** Success, warning, and error colors are calibrated for legibility against light backgrounds, ensuring they meet WCAG AA standards. They are slightly desaturated to prevent "vibration" against the white background while maintaining high urgency.
- **Borders:** #E0E0E0 is used for hair-line dividers (1px) to separate tool windows and editor panes.

## Typography

Typography in this design system prioritizes legibility and information density. 
- **Headings:** Manrope provides a modern, geometric feel for page titles and section headers, adding a touch of personality to the workspace.
- **Interface Text:** Inter is used for all UI labels, inputs, and menus due to its exceptional readability at small sizes and high x-height.
- **Code:** JetBrains Mono is the dedicated typeface for editor blocks and terminal outputs, chosen for its increased character height and clear distinction between similar characters (l, 1, I).
- **Sizing:** The system uses a relatively small base size (14px) to maximize the "information per screen" ratio required for complex technical workflows.

## Layout & Spacing

This design system utilizes a **Fixed-Fluid Hybrid** layout. 
- **Sidebars & Panels:** Utility panels (File Explorer, Debugger) use fixed widths (e.g., 240px to 320px) to maintain a consistent scanning area.
- **Main Editor:** The central work area is fluid, expanding to fill all remaining horizontal and vertical space.
- **Spacing Rhythm:** A 4px baseline grid ensures tight, professional alignment. Most UI components use 8px (space-sm) for internal padding and 12px for edge margins.
- **Density:** High-density spacing is the default. Padding is minimized in lists and trees (e.g., File Explorer) to allow for deep nested hierarchies without excessive scrolling.

## Elevation & Depth

In line with its clean, technical nature, this design system avoids heavy shadows. 
- **Tonal Layering:** Depth is primarily communicated through color shifts. The main editor is #FFFFFF (Level 0), while surrounding panels are #F5F5F5 (Level -1), creating a natural "inset" feel for the code.
- **Outlines:** Instead of shadows, components like popovers, tooltips, and dropdown menus use 1px solid borders (#E0E0E0).
- **Focused Elevation:** Only critical floating elements (Modals or Command Palettes) utilize a soft, high-diffusion shadow (0px 4px 12px rgba(0,0,0,0.08)) to lift them above the interface layers.
- **Active State:** Selection and focus are indicated by the primary blue (#0052CC) rather than depth changes.

## Shapes

The shape language is **Soft (0.25rem / 4px)**. This choice strikes a balance between the precision of sharp corners and the modern feel of rounded UI. 
- **Standard Elements:** Buttons, input fields, and tags use a 4px corner radius.
- **Container Elements:** Tabs and panels remain sharp (0px) when they are docked to the edges of the screen to reinforce the feeling of an integrated toolset.
- **Large Components:** Modals use an 8px (rounded-lg) radius to feel more approachable as distinct global actions.

## Components

- **Buttons:** Primary buttons use a solid #0052CC fill with white text. Secondary buttons use a white background with a #E0E0E0 border and grey text. Tertiary buttons are ghost-style, appearing only as text until hover.
- **Input Fields:** Use a 1px border (#E0E0E0). On focus, the border transitions to #0052CC with a subtle 2px outer glow.
- **Tabs:** Use a "Folder" metaphor. Active tabs have a white background and a 2px top-border in #0052CC, while inactive tabs match the panel background (#F5F5F5).
- **Tree Views (File Explorer):** Use high-density rows (24px height). Selected items use a light blue tint (#E6F0FF) with a solid blue left-edge indicator.
- **Status Bar:** Positioned at the very bottom, using a #0052CC background for high visibility of environment status, or #F5F5F5 for a neutral "idle" state.
- **Terminal:** Uses a #FAFAFA background to distinguish it from the main code editor, maintaining the JetBrains Mono font at a slightly smaller scale (12px).
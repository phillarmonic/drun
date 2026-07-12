# Mermaid pan and zoom component

This directory contains a reusable, dependency-free browser component for
Zensical sites that render Mermaid fences as `.mermaid` elements.

## Install

Copy `docs/assets/components/mermaid-panzoom` into the target project's docs
assets directory, preserving the four files in the component folder.

Register its stylesheet and entry module in `zensical.toml`:

```toml
extra_css = [
  "assets/components/mermaid-panzoom/styles.css",
]

extra_javascript = [
  { path = "assets/components/mermaid-panzoom/index.js", type = "module" },
]
```

If the project already declares `extra_css` or `extra_javascript`, append these
entries to the existing arrays.

## Requirements

- Mermaid fences must render with the `.mermaid` class.
- The browser must support ES modules, `MutationObserver`, and `AbortController`.
- Fullscreen vector rendering loads Mermaid 11 from jsDelivr on first use. If
  the import fails, the component falls back to the diagram already on the page.

## Optional inline callout

The component normally adds a compact `Fullscreen` button to every diagram. To
also show the larger explanatory action row, wrap a diagram in an element with
either `data-mermaid-fullscreen-callout="true"` or the
`mermaid-fullscreen-callout` class.

## Files

- `index.js` handles Zensical navigation and Mermaid block discovery.
- `viewer.js` implements the fullscreen viewer, pan, zoom, and vector rendering.
- `highlights.js` implements node and edge relationship highlighting.
- `styles.css` contains the component styles.

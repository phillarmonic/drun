import { enhanceAllSvgHighlighting } from "./highlights.js";
import { installViewer } from "./viewer.js";

const ENHANCED_ATTR = "data-mermaid-panzoom-bound";
const ACTIONS_ATTR = "data-mermaid-panzoom-actions";
const INDEX_ATTR = "data-mermaid-panzoom-index";
const SHOW_INLINE_ACTIONS_ATTR = "data-mermaid-fullscreen-callout";

const viewer = installViewer();
const sourceByBlock = new WeakMap();
let pageObserver = null;

function shouldShowInlineActions(block) {
  return Boolean(
    block.closest(`[${SHOW_INLINE_ACTIONS_ATTR}="true"], .mermaid-fullscreen-callout`),
  );
}

function openBlock(block) {
  return viewer.openBlock(block, sourceByBlock.get(block) || "");
}

function enhanceMermaidBlock(block, blockIndex = -1) {
  if (!(block instanceof HTMLElement)) return;
  if (block.getAttribute(ENHANCED_ATTR) === "true") return;

  const source = block.querySelector(":scope > code")?.textContent || "";
  if (source) sourceByBlock.set(block, source);
  if (blockIndex >= 0) {
    block.setAttribute(INDEX_ATTR, String(blockIndex));
  }

  block.setAttribute(ENHANCED_ATTR, "true");
  block.classList.add("mermaid-zoomable");
  block.title = "Open fullscreen diagram viewer";

  const trigger = document.createElement("button");
  trigger.type = "button";
  trigger.className = "mermaid-zoom-trigger";
  trigger.textContent = "Fullscreen";
  trigger.setAttribute("aria-label", "Open fullscreen diagram viewer");

  trigger.addEventListener("click", (event) => {
    event.preventDefault();
    event.stopPropagation();
    void openBlock(block);
  });

  block.appendChild(trigger);

  const actionsSibling = block.nextElementSibling;
  const shouldShowActions = shouldShowInlineActions(block);
  if (
    shouldShowActions &&
    (!actionsSibling || actionsSibling.getAttribute(ACTIONS_ATTR) !== "true")
  ) {
    const actions = document.createElement("div");
    actions.className = "mermaid-zoom-actions";
    actions.setAttribute(ACTIONS_ATTR, "true");
    actions.innerHTML = `
      <button type="button" class="mermaid-zoom-actions__button">Open fullscreen diagram</button>
      <span class="mermaid-zoom-actions__hint">Pan with two-finger scroll. Zoom with pinch or Ctrl/Cmd + wheel.</span>
    `;

    actions.querySelector("button").addEventListener("click", (event) => {
      event.preventDefault();
      void openBlock(block);
    });

    block.insertAdjacentElement("afterend", actions);
  } else if (!shouldShowActions && actionsSibling?.getAttribute(ACTIONS_ATTR) === "true") {
    actionsSibling.remove();
  }

  block.addEventListener("click", (event) => {
    if (event.target instanceof HTMLElement && event.target.closest(".mermaid-zoom-trigger")) return;
    void openBlock(block);
  });

  // Install edge highlighting on the rendered SVG
  enhanceAllSvgHighlighting(block);
}

function enhanceAllMermaidBlocks(root = document) {
  const blocks = Array.from(root.querySelectorAll?.(".mermaid") || []);
  if (root instanceof HTMLElement && root.matches(".mermaid")) blocks.unshift(root);
  const indices = new Map(Array.from(document.querySelectorAll(".mermaid"), (block, index) => [block, index]));
  blocks.forEach((block) => enhanceMermaidBlock(block, indices.get(block) ?? -1));
}

function installObserver() {
  pageObserver?.disconnect();
  pageObserver = new MutationObserver((mutations) => {
    const affectedBlocks = new Set();

    for (const mutation of mutations) {
      if (mutation.target instanceof HTMLElement) {
        const parentMermaid = mutation.target.closest(".mermaid");
        if (parentMermaid) affectedBlocks.add(parentMermaid);
      }

      for (const node of mutation.addedNodes) {
        if (!(node instanceof HTMLElement)) continue;
        const closestMermaid = node.closest(".mermaid");
        if (closestMermaid) affectedBlocks.add(closestMermaid);
        if (node.matches(".mermaid")) affectedBlocks.add(node);
        node.querySelectorAll(".mermaid").forEach((block) => affectedBlocks.add(block));
      }
    }

    if (affectedBlocks.size === 0) return;
    const indices = new Map(Array.from(document.querySelectorAll(".mermaid"), (block, index) => [block, index]));
    affectedBlocks.forEach((block) => {
      enhanceMermaidBlock(block, indices.get(block) ?? -1);
      enhanceAllSvgHighlighting(block);
    });
  });

  pageObserver.observe(document.body, { childList: true, subtree: true });
}

function boot() {
  enhanceAllMermaidBlocks();
  installObserver();
}

if (typeof document$ !== "undefined" && document$.subscribe) {
  document$.subscribe(() => {
    window.requestAnimationFrame(boot);
  });
} else {
  document.addEventListener("DOMContentLoaded", boot);
}

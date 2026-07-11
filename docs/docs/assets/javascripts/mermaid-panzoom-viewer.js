import { enhanceAllSvgHighlighting } from "./mermaid-panzoom-highlights.js";

const OVERLAY_ID = "mermaid-panzoom-overlay";
const ENHANCED_ATTR = "data-mermaid-panzoom-bound";
const ACTIONS_ATTR = "data-mermaid-panzoom-actions";
const INDEX_ATTR = "data-mermaid-panzoom-index";
const SHOW_INLINE_ACTIONS_ATTR = "data-mermaid-fullscreen-callout";
const SCALE_MIN = 0.35;
const SCALE_MAX = 4;
const SCALE_STEP = 1.15;
const MERMAID_MODULE_URL = "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs";

let mermaidApiPromise = null;
let mermaidSourcesPromise = null;
let mermaidRenderCounter = 0;

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

async function loadMermaidApi() {
  if (!mermaidApiPromise) {
    mermaidApiPromise = import(MERMAID_MODULE_URL).then((module) => {
      const mermaid = module.default || module;
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        theme: "neutral",
        flowchart: { useMaxWidth: false },
      });
      return mermaid;
    });
  }

  return mermaidApiPromise;
}

async function loadMermaidSources() {
  if (!mermaidSourcesPromise) {
    mermaidSourcesPromise = fetch(window.location.href, { credentials: "same-origin" })
      .then((response) => response.text())
      .then((html) => {
        const doc = new DOMParser().parseFromString(html, "text/html");
        return Array.from(doc.querySelectorAll("pre.mermaid > code"), (code) => code.textContent || "");
      })
      .catch(() => []);
  }

  return mermaidSourcesPromise;
}

async function renderVectorDiagram(source) {
  if (!source) return null;

  const mermaid = await loadMermaidApi();
  const renderId = `mermaid-panzoom-render-${++mermaidRenderCounter}`;
  const { svg, bindFunctions } = await mermaid.render(renderId, source);
  const wrapper = document.createElement("div");
  wrapper.className = "mermaid-panzoom-vector-host";
  wrapper.innerHTML = svg;
  bindFunctions?.(wrapper);
  return wrapper;
}

function createOverlay() {
  const overlay = document.createElement("div");
  overlay.id = OVERLAY_ID;
  overlay.className = "mermaid-panzoom-overlay";
  overlay.hidden = true;
  overlay.innerHTML = `
    <div class="mermaid-panzoom-backdrop" data-close-overlay="true"></div>
    <div class="mermaid-panzoom-shell" role="dialog" aria-modal="true" aria-label="Mermaid diagram viewer">
      <div class="mermaid-panzoom-toolbar">
        <span class="mermaid-panzoom-hint">Scroll to pan. Pinch or Ctrl/Cmd + wheel to zoom.</span>
        <div class="mermaid-panzoom-actions">
          <button type="button" data-action="zoom-out" aria-label="Zoom out">-</button>
          <button type="button" data-action="reset" aria-label="Reset zoom">Reset</button>
          <button type="button" data-action="zoom-in" aria-label="Zoom in">+</button>
          <button type="button" data-action="close" aria-label="Close viewer">Close</button>
        </div>
      </div>
      <div class="mermaid-panzoom-viewport">
        <div class="mermaid-panzoom-stage">
          <div class="mermaid-panzoom-content">
            <div class="mermaid-panzoom-canvas"></div>
          </div>
        </div>
      </div>
    </div>
  `;
  document.body.appendChild(overlay);
  return overlay;
}

function getOverlay() {
  return document.getElementById(OVERLAY_ID) || createOverlay();
}

export function installViewer() {
  const overlay = getOverlay();
  const viewport = overlay.querySelector(".mermaid-panzoom-viewport");
  const stage = overlay.querySelector(".mermaid-panzoom-stage");
  const content = overlay.querySelector(".mermaid-panzoom-content");
  const canvas = overlay.querySelector(".mermaid-panzoom-canvas");

  const state = {
    scale: 1,
    fitScale: 1,
    dragging: false,
    pointerId: null,
    lastX: 0,
    lastY: 0,
    baseWidth: 0,
    baseHeight: 0,
    baseVectorWidth: 0,
    baseVectorHeight: 0,
    mountedNode: null,
    mountedPlaceholder: null,
    mountedParent: null,
    mountedNextSibling: null,
    requestToken: 0,
  };

  function getSubject() {
    return canvas.firstElementChild || canvas;
  }

  function getSubjectSvg() {
    const subject = getSubject();
    if (subject instanceof SVGElement) return subject;
    if (!(subject instanceof HTMLElement)) return null;
    return subject.querySelector("svg");
  }

  function render() {
    const scaledVectorWidth = state.baseVectorWidth * state.scale;
    const scaledVectorHeight = state.baseVectorHeight * state.scale;
    const extraWidth = Math.max(0, state.baseWidth - state.baseVectorWidth);
    const extraHeight = Math.max(0, state.baseHeight - state.baseVectorHeight);
    const subject = getSubject();
    const svg = getSubjectSvg();

    content.style.width = `${scaledVectorWidth + extraWidth}px`;
    content.style.height = `${scaledVectorHeight + extraHeight}px`;

    if (svg) {
      svg.style.width = `${scaledVectorWidth}px`;
      svg.style.height = `${scaledVectorHeight}px`;
    } else if (subject instanceof HTMLElement || subject instanceof SVGElement) {
      subject.style.width = `${state.baseWidth * state.scale}px`;
      subject.style.height = `${state.baseHeight * state.scale}px`;
    }
  }

  function measureBaseSize() {
    const subject = getSubject();
    const svg = getSubjectSvg();
    const rect = subject.getBoundingClientRect();
    const vectorRect = svg?.getBoundingClientRect() || rect;

    state.baseWidth = Math.max(
      1,
      rect.width / state.scale,
      subject.scrollWidth,
      subject.clientWidth,
      subject.offsetWidth,
    );
    state.baseHeight = Math.max(
      1,
      rect.height / state.scale,
      subject.scrollHeight,
      subject.clientHeight,
      subject.offsetHeight,
    );
    state.baseVectorWidth = Math.max(
      1,
      vectorRect.width / state.scale,
      svg?.viewBox?.baseVal?.width || 0,
      svg?.width?.baseVal?.value || 0,
    );
    state.baseVectorHeight = Math.max(
      1,
      vectorRect.height / state.scale,
      svg?.viewBox?.baseVal?.height || 0,
      svg?.height?.baseVal?.value || 0,
    );
  }

  function centerViewport() {
    viewport.scrollLeft = Math.max(0, (viewport.scrollWidth - viewport.clientWidth) / 2);
    viewport.scrollTop = Math.max(0, (viewport.scrollHeight - viewport.clientHeight) / 2);
  }

  function getFitScaleFloor() {
    return Math.min(SCALE_MIN, state.fitScale || SCALE_MIN);
  }

  function fitToViewport() {
    const stageStyles = window.getComputedStyle(stage);
    const paddingX = (Number.parseFloat(stageStyles.paddingLeft) || 0) + (Number.parseFloat(stageStyles.paddingRight) || 0);
    const paddingY = (Number.parseFloat(stageStyles.paddingTop) || 0) + (Number.parseFloat(stageStyles.paddingBottom) || 0);
    const availableWidth = Math.max(1, viewport.clientWidth - paddingX);
    const availableHeight = Math.max(1, viewport.clientHeight - paddingY);
    const widthScale = availableWidth / Math.max(1, state.baseWidth);
    const heightScale = availableHeight / Math.max(1, state.baseHeight);
    state.fitScale = clamp(Math.min(widthScale, heightScale), 0.05, SCALE_MAX);
    state.scale = state.fitScale;
  }

  function resetView() {
    content.style.width = "";
    content.style.height = "";
    const subject = getSubject();
    const svg = getSubjectSvg();
    if (subject instanceof HTMLElement || subject instanceof SVGElement) {
      subject.style.width = "";
      subject.style.height = "";
    }
    if (svg) {
      svg.style.width = "";
      svg.style.height = "";
    }
    window.requestAnimationFrame(() => {
      measureBaseSize();
      fitToViewport();
      render();
      centerViewport();
    });
  }

  function close() {
    if (state.mountedNode && state.mountedPlaceholder && state.mountedParent) {
      state.mountedParent.insertBefore(state.mountedNode, state.mountedNextSibling);
      state.mountedPlaceholder.remove();
      state.mountedNode.classList.remove("mermaid-panzoom-live-host");
    }

    state.mountedNode = null;
    state.mountedPlaceholder = null;
    state.mountedParent = null;
    state.mountedNextSibling = null;
    overlay.hidden = true;
    document.body.classList.remove("mermaid-panzoom-open");
    canvas.replaceChildren();
    content.style.width = "";
    content.style.height = "";
    resetView();
  }

  function open(svgSource) {
    const clone = svgSource.cloneNode(true);
    clone.removeAttribute("width");
    clone.removeAttribute("height");
    canvas.replaceChildren(clone);
    overlay.hidden = false;
    document.body.classList.add("mermaid-panzoom-open");
    resetView();
    // Double-rAF: first rAF lets resetView() finish layout, second ensures
    // geometry (getCTM/getBBox) is fully resolved before building edge maps.
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => enhanceAllSvgHighlighting(canvas, true));
    });
  }

  function openNode(node) {
    canvas.replaceChildren(node);
    overlay.hidden = false;
    document.body.classList.add("mermaid-panzoom-open");
    resetView();
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => enhanceAllSvgHighlighting(canvas, true));
    });
  }

  function openHost(host) {
    const placeholder = document.createElement("div");
    placeholder.className = "mermaid-panzoom-placeholder";
    placeholder.style.display = "none";

    state.mountedNode = host;
    state.mountedPlaceholder = placeholder;
    state.mountedParent = host.parentNode;
    state.mountedNextSibling = host.nextSibling;

    state.mountedParent.insertBefore(placeholder, host);
    host.classList.add("mermaid-panzoom-live-host");
    canvas.replaceChildren(host);
    overlay.hidden = false;
    document.body.classList.add("mermaid-panzoom-open");
    resetView();
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => enhanceAllSvgHighlighting(canvas, true));
    });
  }

  async function openBlock(block, capturedSource = "") {
    const currentToken = ++state.requestToken;
    let source = capturedSource;
    if (!source) {
      const sourceIndex = Number.parseInt(block.getAttribute(INDEX_ATTR) || "", 10);
      if (Number.isInteger(sourceIndex)) {
        const sources = await loadMermaidSources();
        source = sources[sourceIndex] || "";
      }
    }

    if (source) {
      try {
        const rendered = await renderVectorDiagram(source);
        if (currentToken !== state.requestToken || !rendered) return;
        openNode(rendered);
        return;
      } catch (error) {
        console.warn("Mermaid vector render failed, falling back to live host.", error);
      }
    }

    const liveSvg = findRenderedSvg(block);
    if (liveSvg) {
      open(liveSvg);
      return;
    }

    if (currentToken === state.requestToken) {
      openHost(block);
    }
  }

  function zoomAt(clientX, clientY, factor) {
    const previousScale = state.scale;
    const nextScale = clamp(state.scale * factor, getFitScaleFloor(), SCALE_MAX);
    if (nextScale === previousScale) return;

    const rect = viewport.getBoundingClientRect();
    const offsetX = clientX - rect.left;
    const offsetY = clientY - rect.top;
    const contentOriginX = content.offsetLeft;
    const contentOriginY = content.offsetTop;
    const worldX = (viewport.scrollLeft + offsetX - contentOriginX) / previousScale;
    const worldY = (viewport.scrollTop + offsetY - contentOriginY) / previousScale;

    state.scale = nextScale;
    render();
    viewport.scrollLeft = content.offsetLeft + worldX * state.scale - offsetX;
    viewport.scrollTop = content.offsetTop + worldY * state.scale - offsetY;
  }

  function zoomFromCenter(factor) {
    const rect = viewport.getBoundingClientRect();
    zoomAt(rect.left + rect.width / 2, rect.top + rect.height / 2, factor);
  }

  overlay.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) return;

    if (target.dataset.closeOverlay === "true" || target.dataset.action === "close") {
      close();
      return;
    }

    if (target.dataset.action === "zoom-in") {
      zoomFromCenter(SCALE_STEP);
    }

    if (target.dataset.action === "zoom-out") {
      zoomFromCenter(1 / SCALE_STEP);
    }

    if (target.dataset.action === "reset") {
      resetView();
    }
  });

  overlay.addEventListener("wheel", (event) => {
    if (overlay.hidden) return;

    if (event.ctrlKey || event.metaKey) {
      event.preventDefault();
      const factor = event.deltaY < 0 ? SCALE_STEP : 1 / SCALE_STEP;
      zoomAt(event.clientX, event.clientY, factor);
    }
  }, { passive: false });

  viewport.addEventListener("pointerdown", (event) => {
    if (overlay.hidden) return;
    if (event.pointerType === "touch") return;
    if (event.target instanceof HTMLElement && event.target.closest("button")) return;
    state.dragging = true;
    state.pointerId = event.pointerId;
    state.lastX = event.clientX;
    state.lastY = event.clientY;
    viewport.setPointerCapture(event.pointerId);
  });

  viewport.addEventListener("pointermove", (event) => {
    if (!state.dragging || state.pointerId !== event.pointerId) return;
    viewport.scrollLeft -= event.clientX - state.lastX;
    viewport.scrollTop -= event.clientY - state.lastY;
    state.lastX = event.clientX;
    state.lastY = event.clientY;
  });

  function stopDragging(event) {
    if (state.pointerId !== event.pointerId) return;
    state.dragging = false;
    state.pointerId = null;
  }

  viewport.addEventListener("pointerup", stopDragging);
  viewport.addEventListener("pointercancel", stopDragging);
  viewport.addEventListener("pointerleave", stopDragging);

  document.addEventListener("keydown", (event) => {
    if (overlay.hidden) return;
    if (event.key === "Escape") close();
    if (event.key === "+" || event.key === "=") {
      zoomFromCenter(SCALE_STEP);
    }
    if (event.key === "-") {
      zoomFromCenter(1 / SCALE_STEP);
    }
    if (event.key === "0") resetView();
  });

  return { open, openHost, openBlock };
}

function findRenderedSvg(container) {
  if (container.matches("svg")) return container;
  return container.querySelector("svg");
}

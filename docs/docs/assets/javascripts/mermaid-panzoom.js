(function () {
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

  function installViewer() {
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

    async function openBlock(block) {
      const currentToken = ++state.requestToken;
      const sourceIndex = Number.parseInt(block.getAttribute(INDEX_ATTR) || "", 10);
      if (Number.isInteger(sourceIndex)) {
        const sources = await loadMermaidSources();
        const source = sources[sourceIndex];
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

    // Hover detection via mousemove on viewport using bounding box hit testing.
    // This is the primary method since pointer-events CSS can be unreliable in the panzoom canvas.
    let lastHoverEl = null;

    viewport.addEventListener("mousemove", (event) => {
      if (state.dragging) return;
      if (overlay.hidden) return;

      const svgEl = canvas.querySelector("svg[data-edge-highlight]");
      if (!svgEl || !svgEl._edgeHighlight) return;

      const { nodes, edgePaths, highlightNode, highlightEdge, clearHighlights } = svgEl._edgeHighlight;
      const x = event.clientX;
      const y = event.clientY;

      // Check if cursor is over any node
      for (const nodeEl of nodes) {
        const rect = nodeEl.getBoundingClientRect();
        if (x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom) {
          if (lastHoverEl !== nodeEl) {
            clearHighlights();
            lastHoverEl = nodeEl;
            highlightNode(nodeEl);
          }
          return;
        }
      }

      // Check if cursor is near any edge path
      for (let idx = 0; idx < edgePaths.length; idx++) {
        const ep = edgePaths[idx];
        const pathEl = ep.matches("path") ? ep : ep.querySelector("path:not(.edge-hit-area)");
        if (!pathEl) continue;
        // Use a generous bounding box check first
        const rect = ep.getBoundingClientRect();
        if (x >= rect.left - 8 && x <= rect.right + 8 && y >= rect.top - 8 && y <= rect.bottom + 8) {
          // More precise: check distance to path using isPointInStroke with wider stroke
          if (isNearPath(pathEl, x, y, svgEl)) {
            if (lastHoverEl !== ep) {
              clearHighlights();
              lastHoverEl = ep;
              highlightEdge(idx);
            }
            return;
          }
        }
      }

      // Nothing under cursor
      if (lastHoverEl) {
        clearHighlights();
        lastHoverEl = null;
      }
    });

    function isNearPath(pathEl, clientX, clientY, svgEl) {
      // Convert client coordinates to SVG coordinates
      const pt = svgEl.createSVGPoint();
      const ctm = svgEl.getScreenCTM();
      if (!ctm) return false;
      pt.x = clientX;
      pt.y = clientY;
      const svgPt = pt.matrixTransform(ctm.inverse());
      // Check if point is near the stroke (use isPointInStroke with wider stroke)
      const originalWidth = pathEl.style.strokeWidth;
      pathEl.style.strokeWidth = "16px";
      const hit = pathEl.isPointInStroke(svgPt);
      pathEl.style.strokeWidth = originalWidth;
      return hit;
    }

    viewport.addEventListener("mouseleave", () => {
      if (lastHoverEl) {
        const svgEl = canvas.querySelector("svg[data-edge-highlight]");
        if (svgEl && svgEl._edgeHighlight) svgEl._edgeHighlight.clearHighlights();
        lastHoverEl = null;
      }
    });

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

  const viewer = installViewer();

  function findRenderedSvg(container) {
    if (container.matches("svg")) return container;
    return container.querySelector("svg");
  }

  function shouldShowInlineActions(block) {
    return Boolean(
      block.closest(`[${SHOW_INLINE_ACTIONS_ATTR}="true"], .mermaid-fullscreen-callout`),
    );
  }

  // ── Edge highlighting logic ──
  function installEdgeHighlighting(svg, forceReinstall) {
    if (!svg) return;
    if (!forceReinstall && svg.getAttribute("data-edge-highlight") === "true") return;
    svg.setAttribute("data-edge-highlight", "true");

    // Force pointer-events on the SVG and interactive children via inline styles
    svg.style.pointerEvents = "auto";

    // Build a map: nodeId → [edgePath indices]
    const nodes = Array.from(svg.querySelectorAll(".node"));
    const edgePaths = Array.from(svg.querySelectorAll(".edgePath, .edgePaths > path.flowchart-link"));
    const edgeLabels = Array.from(svg.querySelectorAll(".edgeLabel"));

    function getEdgePathElement(edgeEl) {
      if (edgeEl.matches("path")) return edgeEl;
      return edgeEl.querySelector("path:not(.edge-hit-area)");
    }

    // --- Build node-to-edge mapping using SCREEN COORDINATES ---
    // Uses getBoundingClientRect/getScreenCTM which are always reliable
    // regardless of SVG viewBox, CSS transforms, or viewport state.

    function getPathEndpointsScreen(pathEl) {
      try {
        const totalLen = pathEl.getTotalLength();
        if (totalLen === 0) return null;
        const screenCTM = pathEl.getScreenCTM();
        if (!screenCTM) return null;

        function toScreen(localPt) {
          const svgPt = svg.createSVGPoint();
          svgPt.x = localPt.x;
          svgPt.y = localPt.y;
          return svgPt.matrixTransform(screenCTM);
        }

        const startLocal = pathEl.getPointAtLength(0);
        const endLocal = pathEl.getPointAtLength(totalLen);
        const nearStart = pathEl.getPointAtLength(Math.min(10, totalLen * 0.08));
        const nearEnd = pathEl.getPointAtLength(Math.max(totalLen - 10, totalLen * 0.92));

        return {
          start: toScreen(startLocal),
          end: toScreen(endLocal),
          nearStart: toScreen(nearStart),
          nearEnd: toScreen(nearEnd),
        };
      } catch (e) {
        return null;
      }
    }

    function distanceFromPointToRect(point, rect) {
      if (!rect || !point) return Number.POSITIVE_INFINITY;
      const dx = Math.max(rect.left - point.x, 0, point.x - rect.right);
      const dy = Math.max(rect.top - point.y, 0, point.y - rect.bottom);
      return Math.hypot(dx, dy);
    }

    function distanceFromPointToPath(point, pathEl) {
      try {
        const totalLen = pathEl.getTotalLength();
        const screenCTM = pathEl.getScreenCTM();
        if (!totalLen || !screenCTM) return Number.POSITIVE_INFINITY;
        let closestDistance = Number.POSITIVE_INFINITY;
        for (let step = 0; step <= 20; step++) {
          const localPoint = pathEl.getPointAtLength(totalLen * step / 20);
          const svgPoint = svg.createSVGPoint();
          svgPoint.x = localPoint.x;
          svgPoint.y = localPoint.y;
          const screenPoint = svgPoint.matrixTransform(screenCTM);
          closestDistance = Math.min(
            closestDistance,
            Math.hypot(point.x - screenPoint.x, point.y - screenPoint.y),
          );
        }
        return closestDistance;
      } catch (e) {
        return Number.POSITIVE_INFINITY;
      }
    }

    // Build adjacency: node index → [edge indices]
    // and edge index → [node indices]
    const nodeToEdges = new Map(); // nodeIdx → Set of edgeIdx
    const edgeToNodes = new Map(); // edgeIdx → [sourceNodeIdx, targetNodeIdx]
    const nodeToIncomingEdges = new Map(); // nodeIdx → Set of edgeIdx
    const nodeToOutgoingEdges = new Map(); // nodeIdx → Set of edgeIdx

    function addNodeEdge(map, nodeIdx, edgeIdx) {
      if (nodeIdx < 0) return;
      if (!map.has(nodeIdx)) map.set(nodeIdx, new Set());
      map.get(nodeIdx).add(edgeIdx);
    }

    // Get node screen rects (getBoundingClientRect is always accurate)
    const nodeRects = nodes.map((n) => {
      try { return n.getBoundingClientRect(); }
      catch (e) { return null; }
    });
    const nodeKeys = nodes.map((node) => node.id.match(/-flowchart-(.+)-\d+$/)?.[1] || null);

    edgePaths.forEach((ep, edgeIdx) => {
      const pathEl = getEdgePathElement(ep);
      if (!pathEl) return;

      let sourceNodeIdx = -1;
      let targetNodeIdx = -1;

      // Mermaid 11 encodes both endpoint IDs in each flowchart-link ID.
      // Prefer this scale-independent relationship data when available.
      const edgeKey = pathEl.id.match(/-L_(.+)_\d+$/)?.[1];
      if (edgeKey) {
        outer: for (let sourceIdx = 0; sourceIdx < nodeKeys.length; sourceIdx++) {
          if (!nodeKeys[sourceIdx]) continue;
          for (let targetIdx = 0; targetIdx < nodeKeys.length; targetIdx++) {
            if (`${nodeKeys[sourceIdx]}_${nodeKeys[targetIdx]}` === edgeKey) {
              sourceNodeIdx = sourceIdx;
              targetNodeIdx = targetIdx;
              break outer;
            }
          }
        }
      }

      // Older Mermaid versions do not expose endpoint IDs, so retain a
      // geometry fallback for their .edgePath groups.
      if (sourceNodeIdx < 0 || targetNodeIdx < 0) {
        const endpoints = getPathEndpointsScreen(pathEl);
        if (!endpoints) return;
        let closestSourceDistance = Number.POSITIVE_INFINITY;
        let closestTargetDistance = Number.POSITIVE_INFINITY;
        const MARGIN = 40;

        for (let i = 0; i < nodes.length; i++) {
          const rect = nodeRects[i];
          if (!rect || (rect.width === 0 && rect.height === 0)) continue;
          const sourceDistance = Math.min(
            distanceFromPointToRect(endpoints.start, rect),
            distanceFromPointToRect(endpoints.nearStart, rect),
          );
          const targetDistance = Math.min(
            distanceFromPointToRect(endpoints.end, rect),
            distanceFromPointToRect(endpoints.nearEnd, rect),
          );
          if (sourceDistance < closestSourceDistance) {
            closestSourceDistance = sourceDistance;
            sourceNodeIdx = i;
          }
          if (targetDistance < closestTargetDistance) {
            closestTargetDistance = targetDistance;
            targetNodeIdx = i;
          }
        }

        if (closestSourceDistance > MARGIN) sourceNodeIdx = -1;
        if (closestTargetDistance > MARGIN) targetNodeIdx = -1;
      }

      edgeToNodes.set(edgeIdx, [sourceNodeIdx, targetNodeIdx]);

      addNodeEdge(nodeToEdges, sourceNodeIdx, edgeIdx);
      addNodeEdge(nodeToEdges, targetNodeIdx, edgeIdx);
      addNodeEdge(nodeToOutgoingEdges, sourceNodeIdx, edgeIdx);
      addNodeEdge(nodeToIncomingEdges, targetNodeIdx, edgeIdx);
    });

    // Mermaid only emits .edgeLabel elements for labelled edges, so their
    // array indices do not necessarily match the edge array. Associate each
    // label with the closest rendered path instead.
    const edgeToLabels = new Map();
    edgeLabels.forEach((label) => {
      const labelRect = label.getBoundingClientRect();
      const labelPoint = {
        x: labelRect.left + labelRect.width / 2,
        y: labelRect.top + labelRect.height / 2,
      };
      let closestEdgeIdx = -1;
      let closestDistance = Number.POSITIVE_INFINITY;

      edgePaths.forEach((edge, edgeIdx) => {
        const pathEl = getEdgePathElement(edge);
        if (!pathEl) return;
        const distance = distanceFromPointToPath(labelPoint, pathEl);
        if (distance < closestDistance) {
          closestDistance = distance;
          closestEdgeIdx = edgeIdx;
        }
      });

      if (closestEdgeIdx >= 0) {
        if (!edgeToLabels.has(closestEdgeIdx)) edgeToLabels.set(closestEdgeIdx, []);
        edgeToLabels.get(closestEdgeIdx).push(label);
      }
    });

    function highlightNode(nodeEl) {
      const nodeIdx = nodes.indexOf(nodeEl);
      const initialEdgeIndices = nodeToEdges.get(nodeIdx);
      if (!initialEdgeIndices || initialEdgeIndices.size === 0) {
        // Even if no edges found, still highlight the node itself
        svg.classList.add("mermaid-highlight-active");
        nodeEl.classList.add("node-highlighted");
        return;
      }

      svg.classList.add("mermaid-highlight-active");
      const connectedNodeIndices = new Set([nodeIdx]);
      const connectedEdgeIndices = new Set();

      function traverseDirection(edgeMap, endpointPosition) {
        const visitedNodeIndices = new Set([nodeIdx]);
        const pendingNodeIndices = [nodeIdx];
        while (pendingNodeIndices.length > 0) {
          const currentNodeIdx = pendingNodeIndices.pop();
          const edgeIndices = edgeMap.get(currentNodeIdx) || [];
          edgeIndices.forEach((edgeIdx) => {
            connectedEdgeIndices.add(edgeIdx);
            const endpointIdx = (edgeToNodes.get(edgeIdx) || [])[endpointPosition];
            if (endpointIdx < 0 || visitedNodeIndices.has(endpointIdx)) return;
            visitedNodeIndices.add(endpointIdx);
            connectedNodeIndices.add(endpointIdx);
            pendingNodeIndices.push(endpointIdx);
          });
        }
      }

      // Keep the hovered node's lineage and its downstream branch. Traversing
      // each direction independently avoids pulling in sibling branches via a
      // shared ancestor.
      traverseDirection(nodeToIncomingEdges, 0);
      traverseDirection(nodeToOutgoingEdges, 1);

      connectedNodeIndices.forEach((idx) => nodes[idx].classList.add("node-highlighted"));
      connectedEdgeIndices.forEach((idx) => {
        edgePaths[idx].classList.add("edge-highlighted");
        (edgeToLabels.get(idx) || []).forEach((label) => label.classList.add("edge-highlighted"));
      });
    }

    function highlightEdge(edgeIdx) {
      svg.classList.add("mermaid-highlight-active");
      edgePaths[edgeIdx].classList.add("edge-solo-highlight");
      (edgeToLabels.get(edgeIdx) || []).forEach((label) => label.classList.add("edge-solo-highlight"));

      // Highlight connected nodes
      const [srcIdx, tgtIdx] = edgeToNodes.get(edgeIdx) || [-1, -1];
      if (srcIdx >= 0) nodes[srcIdx].classList.add("node-highlighted");
      if (tgtIdx >= 0) nodes[tgtIdx].classList.add("node-highlighted");
    }

    function clearHighlights() {
      svg.classList.remove("mermaid-highlight-active");
      nodes.forEach((n) => n.classList.remove("node-highlighted"));
      edgePaths.forEach((ep) => {
        ep.classList.remove("edge-highlighted");
        ep.classList.remove("edge-solo-highlight");
      });
      edgeLabels.forEach((el) => {
        el.classList.remove("edge-highlighted");
        el.classList.remove("edge-solo-highlight");
      });
    }

    // Force inline pointer-events on all interactive elements
    nodes.forEach((nodeEl) => {
      nodeEl.style.pointerEvents = "all";
      nodeEl.style.cursor = "pointer";
    });

    edgePaths.forEach((ep) => {
      ep.style.pointerEvents = "all";
      ep.style.cursor = "pointer";
      // Add wider hit area for thin edge lines
      const pathEl = getEdgePathElement(ep);
      if (pathEl) {
        const hitArea = pathEl.cloneNode(true);
        hitArea.setAttribute("class", "edge-hit-area");
        hitArea.setAttribute("stroke-width", "14");
        hitArea.setAttribute("stroke", "transparent");
        hitArea.setAttribute("fill", "none");
        hitArea.style.pointerEvents = "stroke";
        pathEl.parentNode.insertBefore(hitArea, pathEl);
      }
    });

    // Use event delegation on the SVG itself via mouseover/mouseout.
    // Skip native handlers when inside the panzoom canvas – the viewport's
    // mousemove handler (getBoundingClientRect-based) is more reliable there
    // because foreignObject HTML content breaks findInteractiveAncestor's
    // namespace traversal, causing it to incorrectly clear highlights.
    const inPanzoomCanvas = svg.closest && svg.closest('.mermaid-panzoom-canvas');

    let currentHighlight = null;

    function findInteractiveAncestor(target) {
      // .closest() doesn't cross SVG/HTML boundaries (foreignObject),
      // so we walk up manually
      let el = target;
      while (el && el !== svg) {
        if (el.classList && el.classList.contains("node")) return { type: "node", el };
        if (el.classList && el.classList.contains("edgePath")) return { type: "edge", el };
        if (el.classList && el.classList.contains("flowchart-link")) return { type: "edge", el };
        // Cross foreignObject boundary
        el = el.parentNode || el.parentElement;
        if (el instanceof HTMLElement && el.closest) {
          const fo = el.closest("foreignObject");
          if (fo) {
            el = fo.parentNode;
          } else {
            el = el.parentElement;
          }
        }
      }
      return null;
    }

    if (!inPanzoomCanvas) {
    svg.addEventListener("mouseover", (event) => {
      const target = event.target;
      const found = findInteractiveAncestor(target);

      if (found && found.type === "node") {
        const nodeEl = found.el;
        if (nodes.includes(nodeEl) && currentHighlight !== nodeEl) {
          clearHighlights();
          currentHighlight = nodeEl;
          highlightNode(nodeEl);
        }
        return;
      }

      if (found && found.type === "edge") {
        const edgeEl = found.el;
        const idx = edgePaths.indexOf(edgeEl);
        if (idx >= 0 && currentHighlight !== edgeEl) {
          clearHighlights();
          currentHighlight = edgeEl;
          highlightEdge(idx);
        }
        return;
      }

      // Not over anything interactive - clear
      if (currentHighlight) {
        clearHighlights();
        currentHighlight = null;
      }
    });

    svg.addEventListener("mouseout", (event) => {
      const related = event.relatedTarget;
      if (!related || !svg.contains(related)) {
        clearHighlights();
        currentHighlight = null;
      }
    });
    } // end if (!inPanzoomCanvas)

    // Expose highlight API on the SVG for viewport-level hit testing
    svg._edgeHighlight = { nodes, edgePaths, edgeLabels, highlightNode, highlightEdge, clearHighlights };
  }

  function enhanceAllSvgHighlighting(root, forceReinstall) {
    if (!root) return;
    const svgs = root.querySelectorAll ? root.querySelectorAll("svg") : [];
    svgs.forEach((s) => installEdgeHighlighting(s, forceReinstall));
    if (root instanceof SVGElement) installEdgeHighlighting(root, forceReinstall);
  }

  function enhanceMermaidBlock(block) {
    if (!(block instanceof HTMLElement)) return;
    if (block.getAttribute(ENHANCED_ATTR) === "true") return;

    const blockIndex = Array.from(document.querySelectorAll(".mermaid")).indexOf(block);
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
      void viewer.openBlock(block);
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
        void viewer.openBlock(block);
      });

      block.insertAdjacentElement("afterend", actions);
    } else if (!shouldShowActions && actionsSibling?.getAttribute(ACTIONS_ATTR) === "true") {
      actionsSibling.remove();
    }

    block.addEventListener("click", (event) => {
      if (event.target instanceof HTMLElement && event.target.closest(".mermaid-zoom-trigger")) return;
      void viewer.openBlock(block);
    });

    // Install edge highlighting on the rendered SVG
    enhanceAllSvgHighlighting(block);
  }

  function enhanceAllMermaidBlocks(root = document) {
    if (root instanceof HTMLElement && root.matches(".mermaid")) {
      enhanceMermaidBlock(root);
    }

    if (root.querySelectorAll) {
      root.querySelectorAll(".mermaid").forEach(enhanceMermaidBlock);
    }
  }

  function installObserver() {
    const observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        if (mutation.target instanceof HTMLElement) {
          const parentMermaid = mutation.target.closest(".mermaid");
          if (parentMermaid) {
            enhanceMermaidBlock(parentMermaid);
            enhanceAllSvgHighlighting(parentMermaid);
          }
        }

        mutation.addedNodes.forEach((node) => {
          if (!(node instanceof HTMLElement)) return;

          const closestMermaid = node.closest(".mermaid");
          if (closestMermaid) {
            enhanceMermaidBlock(closestMermaid);
            enhanceAllSvgHighlighting(closestMermaid);
          }

          if (node.matches(".mermaid")) enhanceMermaidBlock(node);
          enhanceAllMermaidBlocks(node);

          // If an SVG was added (mermaid finished rendering), install highlighting
          if (node.matches("svg") || node.querySelector("svg")) {
            const mermaidParent = node.closest(".mermaid");
            if (mermaidParent) enhanceAllSvgHighlighting(mermaidParent);
          }
        });
      }
    });

    observer.observe(document.body, { childList: true, subtree: true, attributes: false });
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
})();

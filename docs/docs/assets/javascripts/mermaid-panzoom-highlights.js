export function installEdgeHighlighting(svg, forceReinstall = false) {
  if (!svg) return;
  if (!forceReinstall && svg._edgeHighlight) return;
  svg._edgeHighlight?.destroy?.();
  svg.querySelectorAll(".edge-hit-area").forEach((hitArea) => hitArea.remove());
  svg.setAttribute("data-edge-highlight", "true");
  const eventController = new AbortController();

  // Force pointer-events on the SVG and interactive children via inline styles
  svg.style.pointerEvents = "auto";

  // Build a map: nodeId → [edgePath indices]
  const nodes = Array.from(svg.querySelectorAll(".node"));
  const edgePaths = Array.from(svg.querySelectorAll(".edgePath, .edgePaths > path.flowchart-link"));
  const edgeLabels = Array.from(svg.querySelectorAll(".edgeLabel"));
  const nodeIndexByElement = new Map(nodes.map((node, index) => [node, index]));
  const edgeIndexByElement = new Map(edgePaths.map((edge, index) => [edge, index]));

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

  const pathSamples = new Map();

  function getPathSamples(pathEl) {
    if (pathSamples.has(pathEl)) return pathSamples.get(pathEl);
    try {
      const totalLen = pathEl.getTotalLength();
      const screenCTM = pathEl.getScreenCTM();
      if (!totalLen || !screenCTM) return [];
      const samples = [];
      for (let step = 0; step <= 20; step++) {
        const localPoint = pathEl.getPointAtLength(totalLen * step / 20);
        const svgPoint = svg.createSVGPoint();
        svgPoint.x = localPoint.x;
        svgPoint.y = localPoint.y;
        samples.push(svgPoint.matrixTransform(screenCTM));
      }
      pathSamples.set(pathEl, samples);
      return samples;
    } catch (e) {
      return [];
    }
  }

  function distanceFromPointToPath(point, pathEl) {
    return getPathSamples(pathEl).reduce(
      (closest, sample) => Math.min(closest, Math.hypot(point.x - sample.x, point.y - sample.y)),
      Number.POSITIVE_INFINITY,
    );
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
  const nodeIndexByKey = new Map(nodeKeys.map((key, index) => [key, index]).filter(([key]) => key));
  const sourceKeys = [...nodeIndexByKey.keys()].sort((a, b) => b.length - a.length);

  edgePaths.forEach((ep, edgeIdx) => {
    const pathEl = getEdgePathElement(ep);
    if (!pathEl) return;

    let sourceNodeIdx = -1;
    let targetNodeIdx = -1;

    // Mermaid 11 encodes both endpoint IDs in each flowchart-link ID.
    // Prefer this scale-independent relationship data when available.
    const edgeKey = pathEl.id.match(/-L_(.+)_\d+$/)?.[1];
    if (edgeKey) {
      for (const sourceKey of sourceKeys) {
        const prefix = `${sourceKey}_`;
        if (!edgeKey.startsWith(prefix)) continue;
        const targetNodeIdxCandidate = nodeIndexByKey.get(edgeKey.slice(prefix.length));
        if (targetNodeIdxCandidate !== undefined) {
          sourceNodeIdx = nodeIndexByKey.get(sourceKey);
          targetNodeIdx = targetNodeIdxCandidate;
          break;
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
    const nodeIdx = nodeIndexByElement.get(nodeEl) ?? -1;
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

  edgePaths.forEach((ep, edgeIndex) => {
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
      hitArea.dataset.edgeIndex = String(edgeIndex);
      hitArea.style.pointerEvents = "stroke";
      pathEl.parentNode.insertBefore(hitArea, pathEl);
    }
  });

  // One delegated handler covers nodes, edges, hit areas, and foreignObject
  // labels in both inline and fullscreen diagrams.
  let currentHighlight = null;

  function findInteractiveElement(event) {
    for (const element of event.composedPath()) {
      if (element === svg) break;
      if (!(element instanceof Element)) continue;
      if (element.classList.contains("node") && nodeIndexByElement.has(element)) {
        return { type: "node", element };
      }
      if (element.classList.contains("edge-hit-area")) {
        const edgeIndex = Number.parseInt(element.dataset.edgeIndex || "", 10);
        if (Number.isInteger(edgeIndex)) return { type: "edge", element: edgePaths[edgeIndex], edgeIndex };
      }
      if (edgeIndexByElement.has(element)) {
        return { type: "edge", element, edgeIndex: edgeIndexByElement.get(element) };
      }
    }
    return null;
  }

  svg.addEventListener("pointerover", (event) => {
    const found = findInteractiveElement(event);

    if (found?.type === "node") {
      if (currentHighlight !== found.element) {
        clearHighlights();
        currentHighlight = found.element;
        highlightNode(found.element);
      }
      return;
    }

    if (found?.type === "edge") {
      if (currentHighlight !== found.element) {
        clearHighlights();
        currentHighlight = found.element;
        highlightEdge(found.edgeIndex);
      }
      return;
    }

    // Not over anything interactive - clear
    if (currentHighlight) {
      clearHighlights();
      currentHighlight = null;
    }
  }, { signal: eventController.signal });

  svg.addEventListener("pointerleave", () => {
    clearHighlights();
    currentHighlight = null;
  }, { signal: eventController.signal });

  function destroy() {
    eventController.abort();
    svg.querySelectorAll(".edge-hit-area").forEach((hitArea) => hitArea.remove());
    clearHighlights();
    svg.removeAttribute("data-edge-highlight");
    delete svg._edgeHighlight;
  }

  svg._edgeHighlight = { highlightNode, highlightEdge, clearHighlights, destroy };
}

export function enhanceAllSvgHighlighting(root, forceReinstall = false) {
  if (!root) return;
  const svgs = root.querySelectorAll?.("svg") || [];
  svgs.forEach((svg) => installEdgeHighlighting(svg, forceReinstall));
  if (root instanceof SVGElement) installEdgeHighlighting(root, forceReinstall);
}

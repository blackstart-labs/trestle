<script lang="ts">
  import { onMount, tick, untrack } from "svelte";
  import Icon, { type IconName } from "../ui/Icon.svelte";
  import Button from "../ui/Button.svelte";
  import Modal from "../ui/Modal.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import BoardCanvas, { type ShapePreview } from "./BoardCanvas.svelte";
  import {
    colors,
    bounds,
    moveSelection,
    duplicateSelection,
    validateScene,
    download,
    svgExport,
    pngExport,
    whiteboardExport,
    type Board,
    type Scene,
    type BoardObject,
    type ObjectKind,
    type Point,
    type Port,
    type Ink,
  } from "./scene";

  let {
    board,
    workspaceName,
    title = $bindable(),
    scene = $bindable(),
    dirty,
    saving,
    error,
    canEdit,
    canArchive,
    onback,
    onsave,
    onreload,
    ondelete,
  }: {
    board: Board;
    workspaceName: string;
    title: string;
    scene: Scene;
    dirty: boolean;
    saving: boolean;
    error: string;
    canEdit: boolean;
    canArchive: boolean;
    onback: () => void;
    onsave: (archived?: boolean) => void;
    onreload: () => void;
    ondelete?: () => void;
  } = $props();

  let showDeleteConfirm = $state(false);

  type WhiteboardTool =
    | "select"
    | "hand"
    | "rectangle"
    | "diamond"
    | "ellipse"
    | "arrow"
    | "line"
    | "pen"
    | "text"
    | "note"
    | "eraser"
    | "wand";

  let tool = $state<WhiteboardTool>("select");
  let toolLocked = $state(false);
  let previousToolBeforeSpace = $state<WhiteboardTool | null>(null);
  let spaceHeld = $state(false);

  let stage = $state<HTMLDivElement | null>(null);
  let svg = $state<SVGSVGElement | undefined>(undefined);
  let labelInput = $state<HTMLInputElement | null>(null);
  let precisionPoints = $state("");

  let ids = $state<string[]>([]);
  let history = $state<string[]>([]);
  let future = $state<string[]>([]);
  let localError = $state("");
  let clipboard = $state<BoardObject[]>([]);
  let exporting = $state(false);

  let menuOpen = $state(false);
  let propertiesOpen = $state(true);
  let precisionOpen = $state(false);
  let extraToolsOpen = $state(false);
  let sketchy = $state(true);
  let grid = $state(true);

  let textEdit = $state("");
  let textDraft = $state("");
  let textBefore = "";

  let camera = $state({ x: 0, y: 0, zoom: 1 });
  let defaults = $state<{
    color: Ink;
    fill: Ink | "none";
    strokeWidth: number;
    dashed: boolean;
    fontSize: number;
    align: "left" | "center" | "right";
  }>({
    color: "ink",
    fill: "none",
    strokeWidth: 2,
    dashed: false,
    fontSize: 18,
    align: "center",
  });

  interface Gesture {
    type: "pan" | "move" | "resize" | "marquee" | "pen" | "draw_shape" | "wire";
    pointer: number;
    start: Point;
    client: Point;
    scene: Scene;
    ids: string[];
    camera: { x: number; y: number; zoom: number };
    object?: BoardObject;
    handle?: string;
    points?: Point[];
    shapeKind?: "rectangle" | "diamond" | "ellipse" | "note" | "line" | "arrow";
  }

  let gesture = $state<Gesture | null>(null);
  let marquee = $state<{
    x: number;
    y: number;
    width: number;
    height: number;
  } | null>(null);
  let preview = $state<{
    points: Point[];
    color: string;
    width: number;
  } | null>(null);
  let shapePreview = $state<ShapePreview | null>(null);

  const editable = $derived(canEdit && !board.archived);
  const selectedObjects = $derived(
    scene.elements.filter((e) => ids.includes(e.id)),
  );
  const primaryObject = $derived(selectedObjects[0]);

  let width = $state(1200);
  let height = $state(800);

  const viewBox = $derived(
    `${camera.x} ${camera.y} ${width / camera.zoom} ${height / camera.zoom}`,
  );

  function change(next: Scene, record = true): boolean {
    if (!editable) return false;
    if (!validateScene(next)) {
      localError =
        "The drawing exceeds its limits (500 objects, 20000 stroke points). Keep labels within 200 characters.";
      return false;
    }
    if (JSON.stringify(next) === JSON.stringify(scene)) return true;
    if (record) {
      history = [...history.slice(-49), JSON.stringify(scene)];
      future = [];
    }
    scene = next;
    localError = "";
    return true;
  }

  function patchSelected(patch: Partial<BoardObject>) {
    defaults = { ...defaults, ...patch };
    if (!ids.length) return;
    change({
      ...scene,
      elements: scene.elements.map((e) =>
        ids.includes(e.id) && !e.locked ? { ...e, ...patch } : e,
      ),
    });
  }

  function undo() {
    if (!editable || !history.length || gesture) return;
    future = [...future, JSON.stringify(scene)];
    scene = JSON.parse(history.at(-1)!);
    history = history.slice(0, -1);
    ids = ids.filter((id) => scene.elements.some((e) => e.id === id));
  }

  function redo() {
    if (!editable || !future.length || gesture) return;
    history = [...history, JSON.stringify(scene)];
    scene = JSON.parse(future.at(-1)!);
    future = future.slice(0, -1);
    ids = ids.filter((id) => scene.elements.some((e) => e.id === id));
  }

  function remove(targetIds = ids) {
    if (!editable || !targetIds.length) return;
    change({
      ...scene,
      elements: scene.elements.filter(
        (e) =>
          !targetIds.includes(e.id) &&
          !targetIds.includes(e.from || "") &&
          !targetIds.includes(e.to || ""),
      ),
    });
    ids = [];
  }

  function clearCanvas() {
    if (!editable || !scene.elements.length) return;
    change({ ...scene, elements: [] });
    ids = [];
    menuOpen = false;
  }

  function duplicate() {
    if (!editable || !ids.length) return;
    const { scene: next, ids: fresh } = duplicateSelection(scene, ids);
    if (change(next)) ids = fresh;
  }

  function bringForward() {
    if (!editable || !ids.length) return;
    const items = scene.elements.filter((e) => ids.includes(e.id));
    const rest = scene.elements.filter((e) => !ids.includes(e.id));
    change({ ...scene, elements: [...rest, ...items] });
  }

  function sendToBack() {
    if (!editable || !ids.length) return;
    const items = scene.elements.filter((e) => ids.includes(e.id));
    const rest = scene.elements.filter((e) => !ids.includes(e.id));
    change({ ...scene, elements: [...items, ...rest] });
  }

  function addStrokeFromPoints() {
    if (!editable) return;
    const pairs = precisionPoints.trim().split(/\s+/).filter(Boolean);
    const pts: Point[] = [];
    for (const pair of pairs) {
      const [xs, ys] = pair.split(",");
      const x = Number(xs),
        y = Number(ys);
      if (Number.isFinite(x) && Number.isFinite(y)) {
        pts.push({
          x: Math.max(0, Math.min(4000, x)),
          y: Math.max(0, Math.min(4000, y)),
        });
      }
    }
    if (pts.length < 2) {
      localError =
        "Enter at least two coordinate pairs, for example: 100,100 200,160 300,100";
      return;
    }
    const stroke: BoardObject = {
      id: crypto.randomUUID(),
      kind: "stroke",
      x: 0,
      y: 0,
      width: 0,
      height: 0,
      text: "",
      color: defaults.color,
      strokeWidth: defaults.strokeWidth,
      dashed: defaults.dashed,
      points: pts,
    };
    if (change({ ...scene, elements: [...scene.elements, stroke] })) {
      ids = [stroke.id];
      precisionPoints = "";
      localError = "";
    }
  }

  function location(clientX: number, clientY: number): Point {
    const rect = svg
      ? svg.getBoundingClientRect()
      : stage!.getBoundingClientRect();
    return {
      x: camera.x + (clientX - rect.left) / camera.zoom,
      y: camera.y + (clientY - rect.top) / camera.zoom,
    };
  }

  const bounded = (p: Point) => ({
    x: Math.max(0, Math.min(4000, p.x)),
    y: Math.max(0, Math.min(4000, p.y)),
  });

  async function editText(id: string) {
    const node = scene.elements.find((e) => e.id === id);
    if (!editable || !node || node.locked || node.kind === "stroke") return;
    ids = [id];
    textEdit = id;
    textDraft = node.text;
    textBefore = JSON.stringify(scene);
    await tick();
    labelInput?.focus();
    labelInput?.select();
  }

  function finishText(cancel = false) {
    if (!textEdit) return;
    const currentId = textEdit;
    textEdit = "";
    if (cancel) {
      scene = JSON.parse(textBefore);
    } else {
      const el = scene.elements.find((e) => e.id === currentId);
      if (el) el.text = textDraft;
      if (JSON.stringify(scene) !== textBefore) {
        history = [...history.slice(-49), textBefore];
        future = [];
      }
    }
  }

  function topmostAt(p: Point): BoardObject | undefined {
    for (const e of [...scene.elements].reverse()) {
      if (e.kind === "arrow") continue;
      if (e.kind === "stroke" && e.points) {
        const box = bounds([e]);
        if (
          p.x >= box.x - 10 &&
          p.x <= box.x + box.width + 10 &&
          p.y >= box.y - 10 &&
          p.y <= box.y + box.height + 10
        ) {
          return e;
        }
      } else if (
        p.x >= e.x &&
        p.x <= e.x + e.width &&
        p.y >= e.y &&
        p.y <= e.y + e.height
      ) {
        return e;
      }
    }
    return undefined;
  }

  function pointerDown(e: PointerEvent) {
    if (!svg || gesture || (e.button !== 0 && e.button !== 1)) return;
    finishText();
    if (e.pointerType === "mouse") e.preventDefault();
    svg.focus({ preventScroll: true });
    svg.setPointerCapture(e.pointerId);

    const point = location(e.clientX, e.clientY);
    const target = e.target as Element;
    const hitId = target.closest<SVGElement>("[data-object]")?.dataset.object;
    const node = scene.elements.find((n) => n.id === hitId);

    const base: Gesture = {
      type: "marquee",
      pointer: e.pointerId,
      start: point,
      client: { x: e.clientX, y: e.clientY },
      scene: JSON.parse(JSON.stringify(scene)),
      ids: [...ids],
      camera: { ...camera },
    };

    if (tool === "hand" || e.button === 1 || spaceHeld) {
      gesture = { ...base, type: "pan" };
      return;
    }

    if (!editable) {
      if (hitId) ids = [hitId];
      else ids = [];
      return;
    }

    if (tool === "eraser") {
      if (hitId) remove([hitId]);
      else {
        const top = topmostAt(point);
        if (top) remove([top.id]);
      }
      return;
    }

    const handle = target.closest<SVGElement>("[data-handle]");
    if (handle) {
      const obj = scene.elements.find((n) => n.id === handle.dataset.target);
      if (obj && !obj.locked) {
        gesture = {
          ...base,
          type: "resize",
          object: obj,
          handle: handle.dataset.handle,
        };
      }
      return;
    }

    if (tool === "pen") {
      ids = [];
      const pt = bounded(point);
      gesture = { ...base, type: "pen", points: [pt] };
      preview = {
        points: [pt],
        color: defaults.color,
        width: defaults.strokeWidth,
      };
      return;
    }

    if (tool === "wand") {
      ids = [];
      const pt = bounded(point);
      gesture = { ...base, type: "pen", points: [pt] };
      preview = {
        points: [pt],
        color: "danger",
        width: 4,
      };
      return;
    }

    if (
      tool === "rectangle" ||
      tool === "diamond" ||
      tool === "ellipse" ||
      tool === "note" ||
      tool === "line" ||
      tool === "arrow"
    ) {
      ids = [];
      gesture = {
        ...base,
        type: "draw_shape",
        shapeKind: tool,
        start: bounded(point),
      };
      if (tool === "line" || tool === "arrow") {
        shapePreview = {
          kind: tool,
          x: point.x,
          y: point.y,
          width: 0,
          height: 0,
          color: defaults.color,
          fill: defaults.fill,
          strokeWidth: defaults.strokeWidth,
          dashed: defaults.dashed,
          points: [bounded(point), bounded(point)],
        };
      } else {
        shapePreview = {
          kind: tool,
          x: point.x,
          y: point.y,
          width: 0,
          height: 0,
          color: defaults.color,
          fill:
            tool === "note" && defaults.fill === "none"
              ? "warning"
              : defaults.fill,
          strokeWidth: defaults.strokeWidth,
          dashed: defaults.dashed,
        };
      }
      return;
    }

    if (tool === "text") {
      const pt = bounded(point);
      const newText: BoardObject = {
        id: crypto.randomUUID(),
        kind: "text",
        x: pt.x,
        y: pt.y,
        width: 140,
        height: 40,
        text: "Text",
        color: defaults.color,
        fontSize: defaults.fontSize,
        align: defaults.align,
      };
      if (change({ ...scene, elements: [...scene.elements, newText] })) {
        ids = [newText.id];
        void editText(newText.id);
        if (!toolLocked) tool = "select";
      }
      return;
    }

    if (node) {
      if (e.shiftKey) {
        ids = ids.includes(node.id)
          ? ids.filter((id) => id !== node.id)
          : [...ids, node.id];
      } else if (!ids.includes(node.id)) {
        ids = [node.id];
      }
      gesture = { ...base, type: "move", ids: [...ids] };
    } else {
      if (!e.shiftKey) ids = [];
      gesture = { ...base, type: "marquee", ids: e.shiftKey ? [...ids] : [] };
    }
  }

  function pointerMove(e: PointerEvent) {
    const g = gesture;
    if (!g || g.pointer !== e.pointerId) return;
    const p = location(e.clientX, e.clientY);
    const dx = p.x - g.start.x;
    const dy = p.y - g.start.y;

    if (g.type === "pan") {
      camera = {
        ...g.camera,
        x: g.camera.x - (e.clientX - g.client.x) / g.camera.zoom,
        y: g.camera.y - (e.clientY - g.client.y) / g.camera.zoom,
      };
    } else if (g.type === "move") {
      scene = moveSelection(g.scene, g.ids, dx, dy);
    } else if (g.type === "marquee") {
      marquee = {
        x: Math.min(p.x, g.start.x),
        y: Math.min(p.y, g.start.y),
        width: Math.abs(dx),
        height: Math.abs(dy),
      };
    } else if (g.type === "pen" && g.points) {
      if (g.points.length < 2000) {
        g.points.push(bounded(p));
        preview = {
          points: [...g.points],
          color: tool === "wand" ? "danger" : defaults.color,
          width: tool === "wand" ? 4 : defaults.strokeWidth,
        };
      }
    } else if (g.type === "draw_shape") {
      const kind = g.shapeKind!;
      if (kind === "line" || kind === "arrow") {
        shapePreview = {
          kind,
          x: Math.min(g.start.x, p.x),
          y: Math.min(g.start.y, p.y),
          width: Math.abs(dx),
          height: Math.abs(dy),
          color: defaults.color,
          fill: defaults.fill,
          strokeWidth: defaults.strokeWidth,
          dashed: defaults.dashed,
          points: [g.start, bounded(p)],
        };
      } else {
        let w = Math.abs(dx);
        let h = Math.abs(dy);
        if (e.shiftKey) {
          const side = Math.max(w, h);
          w = side;
          h = side;
        }
        const x = dx < 0 ? g.start.x - w : g.start.x;
        const y = dy < 0 ? g.start.y - h : g.start.y;
        shapePreview = {
          kind,
          x,
          y,
          width: Math.max(4, w),
          height: Math.max(4, h),
          color: defaults.color,
          fill:
            kind === "note" && defaults.fill === "none"
              ? "warning"
              : defaults.fill,
          strokeWidth: defaults.strokeWidth,
          dashed: defaults.dashed,
        };
      }
    } else if (g.type === "resize" && g.object) {
      const n = g.object;
      const h = g.handle || "se";
      const left = h.includes("w")
        ? Math.max(0, Math.min(n.x + n.width - 20, n.x + dx))
        : n.x;
      const top = h.includes("n")
        ? Math.max(0, Math.min(n.y + n.height - 20, n.y + dy))
        : n.y;
      const right = h.includes("e")
        ? Math.min(4000, Math.max(n.x + 20, n.x + n.width + dx))
        : n.x + n.width;
      const bottom = h.includes("s")
        ? Math.min(4000, Math.max(n.y + 20, n.y + n.height + dy))
        : n.y + n.height;

      scene = {
        ...g.scene,
        elements: g.scene.elements.map((item) =>
          item.id === n.id
            ? {
                ...item,
                x: left,
                y: top,
                width: right - left,
                height: bottom - top,
              }
            : item,
        ),
      };
    }
  }

  function pointerUp(e: PointerEvent) {
    const g = gesture;
    if (!g || g.pointer !== e.pointerId) return;
    if (svg?.hasPointerCapture(e.pointerId))
      svg.releasePointerCapture(e.pointerId);

    if (e.type === "pointercancel") {
      scene = g.scene;
      gesture = null;
      preview = null;
      shapePreview = null;
      marquee = null;
      return;
    }

    if (tool === "wand") {
      gesture = null;
      shapePreview = null;
      marquee = null;
      setTimeout(() => {
        preview = null;
      }, 400);
      return;
    }

    if (g.type === "pen" && g.points && g.points.length >= 2) {
      const stroke: BoardObject = {
        id: crypto.randomUUID(),
        kind: "stroke",
        x: 0,
        y: 0,
        width: 0,
        height: 0,
        text: "",
        color: defaults.color,
        strokeWidth: defaults.strokeWidth,
        dashed: defaults.dashed,
        points: g.points,
      };
      if (change({ ...g.scene, elements: [...g.scene.elements, stroke] })) {
        ids = [stroke.id];
      }
      if (!toolLocked) tool = "select";
    } else if (g.type === "draw_shape" && shapePreview) {
      const p = location(e.clientX, e.clientY);
      const kind = g.shapeKind!;
      if (kind === "line" || kind === "arrow") {
        const dist = Math.hypot(p.x - g.start.x, p.y - g.start.y);
        if (dist >= 10) {
          const stroke: BoardObject = {
            id: crypto.randomUUID(),
            kind: "stroke",
            x: 0,
            y: 0,
            width: 0,
            height: 0,
            text: "",
            color: defaults.color,
            strokeWidth: defaults.strokeWidth,
            dashed: defaults.dashed,
            points: [g.start, bounded(p)],
          };
          if (change({ ...g.scene, elements: [...g.scene.elements, stroke] })) {
            ids = [stroke.id];
          }
        }
      } else {
        const w = Math.max(20, shapePreview.width);
        const h = Math.max(20, shapePreview.height);
        const shape: BoardObject = {
          id: crypto.randomUUID(),
          kind,
          x: shapePreview.x,
          y: shapePreview.y,
          width: w,
          height: h,
          text: "",
          color: defaults.color,
          fill: shapePreview.fill as Ink | "none",
          strokeWidth: defaults.strokeWidth,
          dashed: defaults.dashed,
        };
        if (change({ ...g.scene, elements: [...g.scene.elements, shape] })) {
          ids = [shape.id];
        }
      }
      if (!toolLocked) tool = "select";
    } else if (g.type === "marquee" && marquee) {
      const m = marquee;
      const found = scene.elements
        .filter(
          (n) =>
            n.kind !== "arrow" &&
            bounds([n]).x >= m.x &&
            bounds([n]).y >= m.y &&
            bounds([n]).x + bounds([n]).width <= m.x + m.width &&
            bounds([n]).y + bounds([n]).height <= m.y + m.height,
        )
        .map((n) => n.id);
      ids = [...new Set([...g.ids, ...found])];
    } else if (
      g.type !== "pan" &&
      JSON.stringify(scene) !== JSON.stringify(g.scene)
    ) {
      history = [...history.slice(-49), JSON.stringify(g.scene)];
      future = [];
    }

    gesture = null;
    preview = null;
    shapePreview = null;
    marquee = null;
  }

  function doubleClick(e: MouseEvent) {
    const point = location(e.clientX, e.clientY);
    const hit = topmostAt(point);
    if (hit) {
      void editText(hit.id);
    } else if (editable) {
      const newText: BoardObject = {
        id: crypto.randomUUID(),
        kind: "text",
        x: point.x,
        y: point.y,
        width: 140,
        height: 40,
        text: "Text",
        color: defaults.color,
        fontSize: defaults.fontSize,
        align: defaults.align,
      };
      if (change({ ...scene, elements: [...scene.elements, newText] })) {
        ids = [newText.id];
        void editText(newText.id);
      }
    }
  }

  function wheel(e: WheelEvent) {
    e.preventDefault();
    if (e.ctrlKey || e.metaKey) {
      const zoomFactor = e.deltaY < 0 ? 1.15 : 0.85;
      const nextZoom = Math.min(3, Math.max(0.2, camera.zoom * zoomFactor));
      const rect = stage!.getBoundingClientRect();
      const pointer = {
        x: camera.x + (e.clientX - rect.left) / camera.zoom,
        y: camera.y + (e.clientY - rect.top) / camera.zoom,
      };
      camera = {
        x: pointer.x - (e.clientX - rect.left) / nextZoom,
        y: pointer.y - (e.clientY - rect.top) / nextZoom,
        zoom: nextZoom,
      };
    } else {
      camera = {
        ...camera,
        x: camera.x + e.deltaX / camera.zoom,
        y: camera.y + e.deltaY / camera.zoom,
      };
    }
  }

  function zoomBy(factor: number) {
    const nextZoom = Math.min(3, Math.max(0.2, camera.zoom * factor));
    camera = { ...camera, zoom: nextZoom };
  }

  function resetZoom() {
    camera = { x: 0, y: 0, zoom: 1 };
  }

  function onKeyDown(e: KeyboardEvent) {
    const target = e.target as HTMLElement;
    if (
      target.tagName === "INPUT" ||
      target.tagName === "TEXTAREA" ||
      target.isContentEditable
    ) {
      if (e.key === "Escape") finishText(true);
      else if (e.key === "Enter" && !e.shiftKey) finishText(false);
      return;
    }

    if (e.key === " " && !spaceHeld) {
      e.preventDefault();
      spaceHeld = true;
      previousToolBeforeSpace = tool;
      tool = "hand";
      return;
    }

    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
      e.preventDefault();
      finishText();
      void tick().then(() => {
        if (editable && dirty) onsave(false);
      });
      return;
    }

    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "z") {
      e.preventDefault();
      if (e.shiftKey) redo();
      else undo();
      return;
    }

    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "y") {
      e.preventDefault();
      redo();
      return;
    }

    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "c") {
      if (ids.length) {
        clipboard = scene.elements.filter((el) => ids.includes(el.id));
      }
      return;
    }

    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "v") {
      if (clipboard.length) {
        const { scene: next, ids: fresh } = duplicateSelection(
          scene,
          clipboard.map((c) => c.id),
          clipboard,
        );
        if (change(next)) ids = fresh;
      }
      return;
    }

    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "d") {
      e.preventDefault();
      duplicate();
      return;
    }

    if (e.key === "Delete" || e.key === "Backspace") {
      e.preventDefault();
      remove();
      return;
    }

    if (e.key === "Escape") {
      if (textEdit) finishText(true);
      else if (menuOpen) menuOpen = false;
      else ids = [];
      return;
    }

    // Single-key tool selection
    const k = e.key.toLowerCase();
    if (k === "1" || k === "v") tool = "select";
    else if (k === "h") tool = "hand";
    else if (k === "2" || k === "r") tool = "rectangle";
    else if (k === "3" || k === "d") tool = "diamond";
    else if (k === "4" || k === "o") tool = "ellipse";
    else if (k === "5" || k === "a") tool = "arrow";
    else if (k === "6" || k === "l") tool = "line";
    else if (k === "7" || k === "p") tool = "pen";
    else if (k === "8" || k === "t") tool = "text";
    else if (k === "9" || k === "n") tool = "note";
    else if (k === "0" || k === "e") tool = "eraser";
  }

  function onKeyUp(e: KeyboardEvent) {
    if (e.key === " " && spaceHeld) {
      spaceHeld = false;
      if (previousToolBeforeSpace) {
        tool = previousToolBeforeSpace;
        previousToolBeforeSpace = null;
      }
    }
  }

  function exportAction(format: "png" | "svg" | "json" | "whiteboard") {
    if (!svg || exporting) return;
    exporting = true;
    try {
      if (format === "json") {
        download(
          JSON.stringify(
            {
              format: "trestle-board",
              version: 1,
              kind: board.kind,
              title,
              scene,
            },
            null,
            2,
          ),
          "application/json",
          `${title.toLowerCase().replace(/[^a-z0-9_-]+/g, "-") || "whiteboard"}.json`,
        );
      } else if (format === "whiteboard") {
        download(
          whiteboardExport(title, scene),
          "application/json",
          `${title.toLowerCase().replace(/[^a-z0-9_-]+/g, "-") || "whiteboard"}.whiteboard.json`,
        );
      } else if (format === "svg") {
        download(
          svgExport(svg),
          "image/svg+xml",
          `${title.toLowerCase().replace(/[^a-z0-9_-]+/g, "-") || "whiteboard"}.svg`,
        );
      } else if (format === "png") {
        void pngExport(svg);
      }
    } finally {
      exporting = false;
      menuOpen = false;
    }
  }

  onMount(() => {
    const observer = new ResizeObserver((entries) => {
      const rect = entries[0]?.contentRect;
      if (rect && rect.width > 0 && rect.height > 0) {
        width = rect.width;
        height = rect.height;
      }
    });
    if (stage) observer.observe(stage);

    const handleBeforeNav = (e: Event) => {
      if (saving) {
        e.preventDefault();
        return;
      }
      if (dirty && editable) {
        finishText();
        onsave(false);
      }
    };
    window.addEventListener("trestle:before-navigate", handleBeforeNav);
    return () => {
      observer.disconnect();
      window.removeEventListener("trestle:before-navigate", handleBeforeNav);
    };
  });
</script>

<svelte:window onkeydown={onKeyDown} onkeyup={onKeyUp} />

<section class="whiteboard-editor" aria-label="Whiteboard editor">
  <h1 class="visually-hidden">{title} — Whiteboard</h1>

  <!-- Top-left Hamburger Menu & Header Details -->
  <div class="top-left-panel">
    <button
      type="button"
      class="btn-quiet btn-icon menu-trigger"
      aria-label="Open menu"
      aria-expanded={menuOpen}
      onclick={() => (menuOpen = !menuOpen)}
    >
      <Icon name="menu" size={20} />
    </button>

    {#if menuOpen}
      <div
        class="whiteboard-menu card"
        role="dialog"
        aria-label="Whiteboard actions"
      >
        <div class="menu-head">
          <label for="board-title" class="field-label">Board title</label>
          <input
            id="board-title"
            class="title-input"
            aria-label="Board title"
            bind:value={title}
            maxlength="120"
            disabled={!editable}
          />
          <p class="workspace-slug">{workspaceName} / Whiteboard</p>
        </div>

        <div class="menu-divider"></div>

        <div class="menu-items">
          {#if canEdit && !board.archived}
            <button
              type="button"
              class="menu-item"
              disabled={saving || !dirty}
              onclick={() => {
                finishText();
                void tick().then(() => onsave(false));
                menuOpen = false;
              }}
            >
              <Icon name="check" size={16} />
              <span>Save board</span>
            </button>
          {/if}

          <button
            type="button"
            class="menu-item"
            onclick={() => exportAction("png")}
          >
            <Icon name="image" size={16} />
            <span>Export image (PNG)</span>
          </button>
          <button
            type="button"
            class="menu-item"
            onclick={() => exportAction("svg")}
          >
            <Icon name="download" size={16} />
            <span>Export vector (SVG)</span>
          </button>
          <button
            type="button"
            class="menu-item"
            onclick={() => exportAction("json")}
          >
            <Icon name="document" size={16} />
            <span>Export JSON draft</span>
          </button>
          <button
            type="button"
            class="menu-item"
            onclick={() => exportAction("whiteboard")}
          >
            <Icon name="download" size={16} />
            <span>Export whiteboard JSON (.whiteboard.json)</span>
          </button>

          <div class="menu-divider"></div>

          <button
            type="button"
            class="menu-item"
            onclick={() => (sketchy = !sketchy)}
          >
            <Icon name="pencil" size={16} />
            <span
              >{sketchy
                ? "Switch to Architect style"
                : "Switch to Hand-drawn style"}</span
            >
          </button>
          <button
            type="button"
            class="menu-item"
            onclick={() => (grid = !grid)}
          >
            <Icon name="grid" size={16} />
            <span>{grid ? "Hide grid" : "Show grid"}</span>
          </button>

          {#if editable}
            <button
              type="button"
              class="menu-item danger-item"
              onclick={clearCanvas}
            >
              <Icon name="trash" size={16} />
              <span>Clear canvas</span>
            </button>
          {/if}

          {#if canArchive && board.id}
            <button
              type="button"
              class="menu-item danger-item"
              onclick={() => {
                menuOpen = false;
                showDeleteConfirm = true;
              }}
            >
              <Icon name="trash" size={16} />
              <span>Delete whiteboard</span>
            </button>
          {/if}

          <div class="menu-divider"></div>

          <button
            type="button"
            class="menu-item"
            aria-label="Back to boards"
            onclick={onback}
          >
            <Icon name="arrow-left" size={16} />
            <span>Back to boards</span>
          </button>
        </div>
      </div>
    {/if}
  </div>

  <!-- Top-Centered Floating Island Dock (Toolbar) -->
  <div class="top-dock-container">
    <header class="whiteboard-dock" aria-label="Drawing tools">
      <!-- Lock Tool Toggle -->
      <button
        type="button"
        class="tool-btn"
        class:active={toolLocked}
        aria-label="Keep selected tool active"
        title="Keep selected tool active after drawing"
        onclick={() => (toolLocked = !toolLocked)}
      >
        <Icon name={toolLocked ? "lock" : "unlock"} size={18} />
      </button>

      <!-- Hand (Pan) -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "hand" || spaceHeld}
        aria-label="Pan canvas"
        title="Hand tool (Space / H)"
        onclick={() => (tool = "hand")}
      >
        <Icon name="hand" size={18} />
      </button>

      <!-- Select -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "select" && !spaceHeld}
        aria-label="Select objects"
        title="Selection pointer (1 / V)"
        onclick={() => (tool = "select")}
      >
        <Icon name="select" size={18} />
      </button>

      <div class="dock-sep"></div>

      <!-- Rectangle -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "rectangle"}
        aria-label="Add rectangle"
        title="Rectangle (2 / R)"
        disabled={!editable}
        onclick={() => (tool = "rectangle")}
      >
        <Icon name="rectangle" size={18} />
      </button>

      <!-- Diamond -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "diamond"}
        aria-label="Add diamond"
        title="Diamond (3 / D)"
        disabled={!editable}
        onclick={() => (tool = "diamond")}
      >
        <Icon name="diamond" size={18} />
      </button>

      <!-- Ellipse -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "ellipse"}
        aria-label="Add ellipse"
        title="Ellipse (4 / O)"
        disabled={!editable}
        onclick={() => (tool = "ellipse")}
      >
        <Icon name="circle" size={18} />
      </button>

      <!-- Arrow -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "arrow"}
        aria-label="Connect shapes"
        title="Arrow (5 / A)"
        disabled={!editable}
        onclick={() => (tool = "arrow")}
      >
        <Icon name="arrow-right" size={18} />
      </button>

      <!-- Line -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "line"}
        aria-label="Line"
        title="Line (6 / L)"
        disabled={!editable}
        onclick={() => (tool = "line")}
      >
        <Icon name="line" size={18} />
      </button>

      <!-- Draw Freehand (Pen) -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "pen"}
        aria-label="Draw freehand"
        title="Draw freehand (7 / P)"
        disabled={!editable}
        onclick={() => (tool = "pen")}
      >
        <Icon name="pencil" size={18} />
      </button>

      <!-- Text -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "text"}
        aria-label="Text"
        title="Text (8 / T)"
        disabled={!editable}
        onclick={() => (tool = "text")}
      >
        <Icon name="type" size={18} />
      </button>

      <!-- Sticky Note -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "note"}
        aria-label="Sticky note"
        title="Sticky note (9 / N)"
        disabled={!editable}
        onclick={() => (tool = "note")}
      >
        <Icon name="note" size={18} />
      </button>

      <!-- Eraser -->
      <button
        type="button"
        class="tool-btn"
        class:active={tool === "eraser"}
        aria-label="Erase object"
        title="Eraser (0 / E)"
        disabled={!editable}
        onclick={() => (tool = "eraser")}
      >
        <Icon name="eraser" size={18} />
      </button>

      <!-- Extra tools / More -->
      <button
        type="button"
        class="tool-btn"
        class:active={extraToolsOpen}
        aria-label="Extra tools"
        title="Extra tools"
        onclick={() => (extraToolsOpen = !extraToolsOpen)}
      >
        <Icon name="more" size={18} />
      </button>
    </header>
  </div>

  <!-- Canvas Movement Hint -->
  <div class="canvas-hint" aria-hidden="true">
    To move canvas, hold <kbd>Scroll wheel</kbd> or <kbd>Space</kbd> while dragging,
    or use the hand tool
  </div>

  <!-- Right Floating Properties Palette (Always visible) -->
  <aside class="whiteboard-properties card" aria-label="Palette and properties">
    <div class="prop-head">
      <span class="prop-title">Palette & Properties</span>
    </div>

    <div class="prop-body">
      <!-- Stroke Color -->
      <div class="prop-section">
        <span class="prop-label">Stroke</span>
        <div class="swatches" role="radiogroup" aria-label="Stroke color">
          {#each colors as c}
            <button
              type="button"
              class="swatch"
              class:selected={(primaryObject?.color || defaults.color) === c}
              style:--swatch-color={`var(${c === "ink" ? "--text" : `--${c}`})`}
              aria-label={`${c} stroke`}
              onclick={() => patchSelected({ color: c })}
            ></button>
          {/each}
        </div>
      </div>

      <!-- Background / Fill -->
      <div class="prop-section">
        <span class="prop-label">Background</span>
        <div class="swatches" role="radiogroup" aria-label="Fill color">
          <button
            type="button"
            class="swatch swatch-none"
            class:selected={(primaryObject?.fill ?? defaults.fill) === "none"}
            aria-label="No fill"
            onclick={() => patchSelected({ fill: "none" })}
          ></button>
          {#each colors as c}
            <button
              type="button"
              class="swatch"
              class:selected={(primaryObject?.fill ?? defaults.fill) === c}
              style:--swatch-color={`var(${c === "ink" ? "--selected" : `--${c}-soft`})`}
              aria-label={`${c} fill`}
              onclick={() => patchSelected({ fill: c })}
            ></button>
          {/each}
        </div>
      </div>

      <!-- Stroke Width -->
      <div class="prop-section">
        <span class="prop-label">Stroke width</span>
        <div class="segmented-control" role="group" aria-label="Stroke width">
          {#each [2, 4, 6] as w}
            <button
              type="button"
              class="seg-btn"
              class:active={(primaryObject?.strokeWidth ||
                defaults.strokeWidth) === w}
              aria-label={`${w} pixel stroke`}
              title={`${w}px`}
              onclick={() => patchSelected({ strokeWidth: w })}
            >
              <span class="line-sample" style:--line-weight={`${w}px`}></span>
            </button>
          {/each}
        </div>
      </div>

      <!-- Stroke Style -->
      <div class="prop-section">
        <span class="prop-label">Stroke style</span>
        <div class="segmented-control" role="group" aria-label="Stroke style">
          <button
            type="button"
            class="seg-btn"
            class:active={!(primaryObject?.dashed ?? defaults.dashed)}
            onclick={() => patchSelected({ dashed: false })}
          >
            Solid
          </button>
          <button
            type="button"
            class="seg-btn"
            class:active={!!(primaryObject?.dashed ?? defaults.dashed)}
            onclick={() => patchSelected({ dashed: true })}
          >
            Dashed
          </button>
        </div>
      </div>

      <!-- Font Size & Align (for text and note) -->
      {#if !primaryObject || primaryObject.kind === "text" || primaryObject.kind === "note"}
        <div class="prop-section">
          <span class="prop-label">Text size</span>
          <div class="segmented-control" role="group" aria-label="Text size">
            {#each [14, 18, 24, 32] as s}
              <button
                type="button"
                class="seg-btn"
                class:active={(primaryObject?.fontSize || defaults.fontSize) ===
                  s}
                onclick={() => patchSelected({ fontSize: s })}
              >
                {s === 14 ? "S" : s === 18 ? "M" : s === 24 ? "L" : "XL"}
              </button>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Actions on selected elements -->
      {#if selectedObjects.length}
        <div class="prop-section actions-section">
          <span class="prop-label">Actions</span>
          <div class="action-buttons">
            <button
              type="button"
              class="btn-quiet btn-compact"
              onclick={duplicate}
              title="Duplicate (Ctrl+D)"
            >
              <Icon name="copy" size={14} />
              <span>Duplicate</span>
            </button>
            <button
              type="button"
              class="btn-quiet btn-compact"
              onclick={bringForward}
              title="Bring forward"
            >
              <Icon name="arrow-up" size={14} />
              <span>Front</span>
            </button>
            <button
              type="button"
              class="btn-quiet btn-compact"
              onclick={sendToBack}
              title="Send to back"
            >
              <Icon name="arrow-down" size={14} />
              <span>Back</span>
            </button>
            <button
              type="button"
              class="btn-quiet btn-compact danger-btn"
              onclick={() => remove()}
              title="Delete (Del)"
            >
              <Icon name="trash" size={14} />
              <span>Delete</span>
            </button>
          </div>
        </div>
      {/if}

      <!-- Precision Coordinates Accordion (Preserves e2e test compatibility) -->
      <details class="precision-drawer" bind:open={precisionOpen}>
        <summary>Draw with coordinates</summary>
        <div class="precision-content">
          <label for="wb-stroke-pts" class="visually-hidden"
            >Stroke points</label
          >
          <input
            id="wb-stroke-pts"
            class="precision-input"
            aria-label="Stroke points"
            placeholder="100,100 200,160 300,100"
            bind:value={precisionPoints}
            disabled={!editable}
          />
          <Button
            variant="secondary"
            disabled={!editable || !precisionPoints.trim()}
            onclick={addStrokeFromPoints}
          >
            Add stroke from points
          </Button>
        </div>
      </details>
    </div>
  </aside>

  <!-- Bottom-Left Floating Zoom & History Island -->
  <footer
    class="whiteboard-zoom-dock"
    aria-label="Canvas navigation and history"
  >
    <button
      type="button"
      class="zoom-btn"
      aria-label="Zoom out"
      title="Zoom out"
      onclick={() => zoomBy(0.8)}
    >
      <Icon name="zoom-out" size={16} />
    </button>
    <button
      type="button"
      class="zoom-display"
      title="Reset zoom to 100%"
      onclick={resetZoom}
    >
      {Math.round(camera.zoom * 100)}%
    </button>
    <button
      type="button"
      class="zoom-btn"
      aria-label="Zoom in"
      title="Zoom in"
      onclick={() => zoomBy(1.25)}
    >
      <Icon name="zoom-in" size={16} />
    </button>

    <div class="dock-sep"></div>

    <button
      type="button"
      class="zoom-btn"
      aria-label="Undo"
      title="Undo (Ctrl+Z)"
      disabled={!editable || !history.length}
      onclick={undo}
    >
      <Icon name="undo" size={16} />
    </button>
    <button
      type="button"
      class="zoom-btn"
      aria-label="Redo"
      title="Redo (Ctrl+Y)"
      disabled={!editable || !future.length}
      onclick={redo}
    >
      <Icon name="redo" size={16} />
    </button>
  </footer>

  <!-- Save State Indicator & Error Banner -->
  <div class="status-overlay">
    <span class="save-state" role="status">
      <Icon name={saving ? "clock" : dirty ? "circle" : "check"} size={14} />
      <span>
        {board.archived
          ? "Archived"
          : saving
            ? "Saving…"
            : dirty
              ? "Unsaved changes"
              : "All changes saved"}
      </span>
    </span>
    {#if error || localError}
      <div class="whiteboard-error">
        <Feedback tone="danger">{error || localError}</Feedback>
        {#if error && board.id}
          <Button disabled={saving} onclick={onreload}
            >Reload saved board</Button
          >
        {/if}
      </div>
    {/if}
  </div>

  <!-- Interactive Full-Bleed Canvas -->
  <div class="canvas-stage" bind:this={stage} role="presentation">
    <BoardCanvas
      bind:svg
      {scene}
      selectedIds={ids}
      {viewBox}
      scale={camera.zoom}
      {grid}
      {sketchy}
      {marquee}
      {preview}
      {shapePreview}
      onpointerdown={pointerDown}
      onpointermove={pointerMove}
      onpointerup={pointerUp}
      ondblclick={doubleClick}
      onwheel={wheel}
    />

    <!-- Inline Text Editor Overlay -->
    {#if textEdit && primaryObject}
      <div
        class="inline-text-box"
        style:left={`${(primaryObject.x - camera.x) * camera.zoom}px`}
        style:top={`${(primaryObject.y - camera.y) * camera.zoom}px`}
        style:width={`${primaryObject.width * camera.zoom}px`}
        style:min-height={`${primaryObject.height * camera.zoom}px`}
      >
        <input
          bind:this={labelInput}
          class="inline-text-input"
          bind:value={textDraft}
          oninput={() => {
            const el = scene.elements.find((e) => e.id === textEdit);
            if (el) el.text = textDraft;
          }}
          onblur={() => finishText(false)}
        />
      </div>
    {/if}
  </div>
</section>

{#if showDeleteConfirm}
  <Modal
    title="Delete whiteboard"
    description={`Are you sure you want to delete “${title}”? This action cannot be undone.`}
    onclose={() => (showDeleteConfirm = false)}
  >
    <div class="dialog-actions">
      <Button variant="quiet" onclick={() => (showDeleteConfirm = false)}
        >Cancel</Button
      >
      <Button
        variant="danger"
        onclick={() => {
          showDeleteConfirm = false;
          ondelete?.();
        }}>Delete whiteboard</Button
      >
    </div>
  </Modal>
{/if}

<style>
  .whiteboard-editor {
    position: relative;
    width: 100%;
    height: 100%;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    background: var(--surface);
    color: var(--text);
    user-select: none;
    -webkit-user-select: none;
    outline: none;
  }

  .canvas-stage {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    cursor: default;
    outline: none;
  }

  /* Top-left Menu */
  .top-left-panel {
    position: absolute;
    top: var(--space-4);
    left: var(--space-4);
    z-index: var(--layer-overlay);
  }

  .menu-trigger {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-md);
    background: var(--surface-float);
    border: 1px solid var(--border);
    box-shadow: var(--shadow-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text);
  }

  .whiteboard-menu {
    position: absolute;
    top: 48px;
    left: 0;
    width: 260px;
    background: var(--surface-float);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-overlay);
    padding: var(--space-3);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .menu-head {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field-label {
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .title-input {
    width: 100%;
    background: var(--surface-raised);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    padding: 6px 8px;
    font: inherit;
    font-weight: 500;
    color: var(--text);
  }

  .workspace-slug {
    font-size: var(--text-xs);
    color: var(--text-secondary);
    margin: 0;
  }

  .menu-divider {
    height: 1px;
    background: var(--line);
    margin: 4px 0;
  }

  .menu-items {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 100%;
    padding: 8px 10px;
    background: transparent;
    border: none;
    border-radius: var(--radius-sm);
    color: var(--text);
    font-size: var(--text-sm);
    cursor: pointer;
    text-align: left;
  }

  .menu-item:hover:not(:disabled) {
    background: var(--selected);
  }

  .danger-item {
    color: var(--danger);
  }

  /* Top-Centered Floating Island Dock (Toolbar) */
  .top-dock-container {
    position: absolute;
    top: var(--space-4);
    left: 50%;
    transform: translateX(-50%);
    z-index: var(--layer-overlay);
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .whiteboard-dock {
    background: var(--surface-float);
    border: 1px solid var(--border);
    border-radius: var(--radius-pill);
    box-shadow: var(--shadow-overlay);
    padding: 4px 8px;
    display: flex;
    align-items: center;
    gap: 3px;
  }

  .tool-btn {
    width: 36px;
    height: 36px;
    border-radius: var(--radius-pill);
    border: none;
    background: transparent;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
  }

  .tool-btn:hover:not(:disabled) {
    background: var(--selected);
    color: var(--text);
  }

  .tool-btn.active {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .dock-sep {
    width: 1px;
    height: 20px;
    background: var(--line);
    margin: 0 4px;
  }

  /* Movement Hint below dock */
  .canvas-hint {
    position: absolute;
    top: calc(var(--space-4) + 52px);
    left: 50%;
    transform: translateX(-50%);
    z-index: var(--layer-sticky);
    font-size: var(--text-xs);
    color: var(--text-muted);
    pointer-events: none;
    text-align: center;
    white-space: nowrap;
  }

  .canvas-hint kbd {
    background: var(--surface-raised);
    border: 1px solid var(--line);
    border-radius: 4px;
    padding: 1px 5px;
    font-size: 11px;
    font-family: var(--font-mono);
    color: var(--text-secondary);
  }

  /* Right Floating Properties Palette */
  .whiteboard-properties {
    position: absolute;
    top: calc(var(--space-4) + 16px);
    right: var(--space-4);
    z-index: var(--layer-overlay);
    width: 240px;
    background: var(--surface-float);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-overlay);
    padding: var(--space-3);
    max-height: calc(100vh - 120px);
    overflow-y: auto;
  }

  .prop-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-2);
  }

  .prop-title {
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
  }

  .prop-body {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .prop-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .prop-label {
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .swatches {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .swatch {
    width: 22px;
    height: 22px;
    border-radius: 4px;
    background: var(--swatch-color);
    border: 1px solid var(--line);
    cursor: pointer;
  }

  .swatch.selected {
    outline: 2px solid var(--accent);
    outline-offset: 1px;
  }

  .swatch-none {
    background: repeating-linear-gradient(
      45deg,
      transparent,
      transparent 3px,
      var(--line) 3px,
      var(--line) 4px
    );
  }

  .segmented-control {
    display: flex;
    background: var(--surface-raised);
    border-radius: var(--radius-sm);
    padding: 2px;
    gap: 2px;
  }

  .seg-btn {
    flex: 1;
    padding: 4px 6px;
    border: none;
    border-radius: calc(var(--radius-sm) - 2px);
    background: transparent;
    color: var(--text-secondary);
    font-size: var(--text-xs);
    font-weight: 500;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .seg-btn.active {
    background: var(--surface-float);
    color: var(--text);
    box-shadow: var(--shadow-sm);
  }

  .line-sample {
    width: 18px;
    height: var(--line-weight);
    background: currentColor;
    border-radius: 1px;
  }

  .action-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .btn-compact {
    padding: 4px 8px;
    font-size: var(--text-xs);
    display: flex;
    align-items: center;
    gap: 4px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--line);
  }

  .danger-btn {
    color: var(--danger);
  }

  /* Precision Drawer */
  .precision-drawer summary {
    font-size: var(--text-xs);
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px 0;
  }

  .precision-content {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 6px;
  }

  .precision-input {
    width: 100%;
    background: var(--surface-raised);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    padding: 4px 6px;
    font-size: var(--text-xs);
    font-family: var(--font-mono);
    color: var(--text);
  }

  /* Bottom-Left Floating Zoom Island */
  .whiteboard-zoom-dock {
    position: absolute;
    bottom: var(--space-4);
    left: var(--space-4);
    z-index: var(--layer-overlay);
    background: var(--surface-float);
    border: 1px solid var(--border);
    border-radius: var(--radius-pill);
    box-shadow: var(--shadow-overlay);
    padding: 4px 8px;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .zoom-btn {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
  }

  .zoom-btn:hover:not(:disabled) {
    background: var(--selected);
    color: var(--text);
  }

  .zoom-display {
    border: none;
    background: transparent;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text);
    padding: 0 4px;
    cursor: pointer;
  }

  /* Save State Indicator */
  .status-overlay {
    position: absolute;
    bottom: var(--space-4);
    right: var(--space-4);
    z-index: var(--layer-overlay);
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--space-2);
  }

  .save-state {
    font-size: var(--text-xs);
    color: var(--text-muted);
    background: var(--surface-float);
    border: 1px solid var(--line);
    border-radius: var(--radius-pill);
    padding: 4px 10px;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  /* Inline Text Box */
  .inline-text-box {
    position: absolute;
    pointer-events: auto;
  }

  .inline-text-input {
    width: 100%;
    height: 100%;
    background: transparent;
    border: 1px dashed var(--accent);
    color: var(--text);
    font: inherit;
    text-align: center;
    outline: none;
  }

  @media (max-width: 60rem) {
    .whiteboard-dock {
      top: auto;
      bottom: var(--space-4);
      left: 50%;
      transform: translateX(-50%);
      width: calc(100% - 32px);
      overflow-x: auto;
      justify-content: flex-start;
    }

    .whiteboard-zoom-dock {
      bottom: calc(var(--space-4) + 56px);
    }

    .canvas-hint {
      display: none;
    }

    .whiteboard-properties {
      top: 60px;
      right: var(--space-4);
      left: auto;
    }
  }
</style>

<script lang="ts">
  import { onMount, tick, untrack } from "svelte";
  import Icon, { type IconName } from "../ui/Icon.svelte";
  import Button from "../ui/Button.svelte";
  import Select from "../ui/Select.svelte";
  import Modal from "../ui/Modal.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import BoardCanvas from "./BoardCanvas.svelte";
  import {
    colors,
    kinds,
    bounds,
    moveSelection,
    duplicateSelection,
    moveObject,
    validateScene,
    download,
    svgExport,
    pngExport,
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
  } = $props();

  const catalog: {
    kind: ObjectKind;
    label: string;
    icon: IconName;
    section: string;
  }[] = [
    {
      kind: "rectangle",
      label: "Process",
      icon: "rectangle",
      section: "Flowchart",
    },
    {
      kind: "diamond",
      label: "Decision",
      icon: "diamond",
      section: "Flowchart",
    },
    {
      kind: "terminator",
      label: "Start / end",
      icon: "terminator",
      section: "Flowchart",
    },
    {
      kind: "document",
      label: "Document",
      icon: "document",
      section: "Flowchart",
    },
    {
      kind: "database",
      label: "Database",
      icon: "database",
      section: "Flowchart",
    },
    {
      kind: "parallelogram",
      label: "Input / output",
      icon: "parallelogram",
      section: "Flowchart",
    },
    { kind: "ellipse", label: "Ellipse", icon: "circle", section: "Standard" },
    { kind: "text", label: "Text", icon: "pencil", section: "Standard" },
    { kind: "note", label: "Sticky note", icon: "note", section: "Standard" },
    { kind: "frame", label: "Frame", icon: "frame", section: "Containers" },
    {
      kind: "swimlane",
      label: "Swimlanes",
      icon: "swimlane",
      section: "Containers",
    },
  ];
  type Tool = "select" | "hand" | "connector" | "pen" | "eraser";
  let tool = $state<Tool>("select");
  let ids = $state<string[]>([]);
  let history = $state<string[]>([]);
  let future = $state<string[]>([]);
  let clipboard = $state<BoardObject[]>([]);
  let localError = $state("");
  let library = $state(true);
  let properties = $state(false);
  let libraryTab = $state<"shapes" | "objects">("shapes");
  let query = $state("");
  let grid = $state(true);
  let snap = $state(true);
  let templates = $state(false);
  let stage = $state<HTMLDivElement>();
  let svg = $state<SVGSVGElement>();
  let width = $state(1000);
  let height = $state(650);
  let camera = $state({ x: -60, y: -60, zoom: 0.75 });
  const initialKind = untrack(() => board.kind);
  let defaults = $state<Partial<BoardObject>>({
    color: initialKind === "diagram" ? "accent" : "ink",
    fill: initialKind === "diagram" ? "accent" : "none",
    strokeWidth: 2,
    fontSize: 18,
    align: "center",
    route: "elbow",
    heads: "end",
  });
  let from = $state("");
  let fromPort = $state<Port>("auto");
  let to = $state("");
  let strokeInput = $state("");
  let preview = $state<Point[]>([]);
  let marquee = $state<{
    x: number;
    y: number;
    width: number;
    height: number;
  } | null>(null);
  let wire = $state<{ from: Point; to: Point } | null>(null);
  let textEdit = $state("");
  let textDraft = $state("");
  let textBefore = "";
  let labelInput = $state<HTMLTextAreaElement>();
  let exporting = $state(false);
  type Gesture = {
    type: "move" | "resize" | "pen" | "pan" | "marquee" | "wire";
    pointer: number;
    start: Point;
    client: Point;
    scene: Scene;
    ids: string[];
    camera: typeof camera;
    object?: BoardObject;
    handle?: string;
    points?: Point[];
    from?: string;
    port?: Port;
  };
  let gesture = $state.raw<Gesture | null>(null);
  const editable = $derived(
    canEdit && !board.archived && !saving && !exporting,
  );
  const selected = $derived(scene.elements.filter((e) => ids.includes(e.id)));
  const object = $derived(selected.at(-1));
  const format = $derived(object || defaults);
  const shapes = $derived(
    scene.elements.filter((e) => e.kind !== "arrow" && e.kind !== "stroke"),
  );
  const editedObject = $derived(scene.elements.find((e) => e.id === textEdit));
  const hint = $derived(
    tool === "connector"
      ? from
        ? "Choose the destination shape, or drag between connection points."
        : "Choose two shapes, or drag between connection points."
      : tool === "hand"
        ? "Drag to pan. Use the wheel to move; Ctrl/⌘ + wheel zooms."
        : tool === "pen"
          ? "Draw with a mouse, pen or touch. Each stroke can be undone."
          : tool === "eraser"
            ? "Select an object to erase it. Undo restores it."
            : selected.length
              ? `${selected.length} selected · Drag to move · Double-click to edit text · Shift-click to add to selection`
              : "Drag a shape onto the page, or choose one from the library. Drag the page to select objects.",
  );

  function change(next: Scene, record = true) {
    if (!editable) return false;
    if (!validateScene(next)) {
      localError =
        "The drawing exceeds its bounds or limits (500 objects, 20000 stroke points). Keep labels within 200 characters.";
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
  function patchSelected(patch: Partial<BoardObject>, unlock = false) {
    if (!ids.length) {
      defaults = { ...defaults, ...patch };
      return;
    }
    change({
      ...scene,
      elements: scene.elements.map((e) =>
        ids.includes(e.id) && (!e.locked || unlock) ? { ...e, ...patch } : e,
      ),
    });
  }
  function undo() {
    if (!editable || !history.length || gesture) return;
    future = [...future, JSON.stringify(scene)];
    scene = JSON.parse(history.at(-1)!);
    history = history.slice(0, -1);
  }
  function redo() {
    if (!editable || !future.length || gesture) return;
    history = [...history, JSON.stringify(scene)];
    scene = JSON.parse(future.at(-1)!);
    future = future.slice(0, -1);
  }
  const snapped = (n: number) =>
    snap ? Math.round(n / 10) * 10 : Math.round(n);
  function location(clientX: number, clientY: number): Point {
    const rect = stage!.getBoundingClientRect();
    return {
      x: camera.x + (clientX - rect.left) / camera.zoom,
      y: camera.y + (clientY - rect.top) / camera.zoom,
    };
  }
  const bounded = (p: Point) => ({
    x: Math.max(0, Math.min(4000, p.x)),
    y: Math.max(0, Math.min(4000, p.y)),
  });
  function make(kind: ObjectKind, at?: Point): BoardObject {
    const wide = kind === "frame" || kind === "swimlane";
    const w = wide ? 720 : kind === "note" || kind === "diamond" ? 180 : 220;
    const h = wide
      ? 480
      : kind === "note" || kind === "diamond"
        ? 160
        : kind === "text"
          ? 60
          : 100;
    const center = at || {
      x: camera.x + width / camera.zoom / 2,
      y: camera.y + height / camera.zoom / 2,
    };
    return {
      id: crypto.randomUUID(),
      kind,
      x: Math.max(0, Math.min(4000 - w, snapped(center.x - w / 2))),
      y: Math.max(0, Math.min(4000 - h, snapped(center.y - h / 2))),
      width: w,
      height: h,
      text:
        kind === "text"
          ? "Text"
          : kind === "frame"
            ? "Frame"
            : kind === "swimlane"
              ? "Lane 1 / Lane 2 / Lane 3"
              : "",
      color: "ink",
      ...defaults,
      ...(wide ? { fill: "none", color: "ink", align: "left" } : {}),
      ...(kind === "note" ? { fill: "warning", color: "warning" } : {}),
    };
  }
  function add(kind: ObjectKind, at?: Point) {
    if (!editable) return;
    finishText();
    const e = make(kind, at);
    if (
      change({
        ...scene,
        elements:
          kind === "frame" || kind === "swimlane"
            ? [e, ...scene.elements]
            : [...scene.elements, e],
      })
    ) {
      ids = [e.id];
      tool = "select";
      if (kind === "text") void editText(e.id);
    }
    if (matchMedia("(max-width: 60rem)").matches) library = false;
  }
  function choose(id: string, extend = false) {
    const e = scene.elements.find((n) => n.id === id);
    const group = e?.group
      ? scene.elements.filter((n) => n.group === e.group).map((n) => n.id)
      : [id];
    ids = extend
      ? group.every((n) => ids.includes(n))
        ? ids.filter((n) => !group.includes(n))
        : [...new Set([...ids, ...group])]
      : group;
  }
  function remove(targets = ids, ask = true) {
    const removable = scene.elements
      .filter((e) => targets.includes(e.id) && !e.locked)
      .map((e) => e.id);
    if (
      !removable.length ||
      (ask &&
        !confirm(
          `Remove ${removable.length} selected object(s) and connected lines? Undo can restore them in this editing session.`,
        ))
    )
      return;
    if (
      change({
        ...scene,
        elements: scene.elements.filter(
          (e) =>
            !removable.includes(e.id) &&
            !removable.includes(e.from || "") &&
            !removable.includes(e.to || ""),
        ),
      })
    )
      ids = [];
  }
  function connect(
    a = from,
    b = to,
    start: Port = fromPort,
    end: Port = "auto",
  ) {
    if (!a || !b || a === b) return;
    const edge: BoardObject = {
      id: crypto.randomUUID(),
      kind: "arrow",
      x: 0,
      y: 0,
      width: 0,
      height: 0,
      text: "",
      color: defaults.color || "ink",
      strokeWidth: defaults.strokeWidth || 2,
      route: defaults.route || "elbow",
      heads: defaults.heads || "end",
      from: a,
      to: b,
      fromPort: start,
      toPort: end,
    };
    if (change({ ...scene, elements: [...scene.elements, edge] })) {
      ids = [edge.id];
      from = "";
      to = "";
      wire = null;
    }
  }
  function duplicate(source = scene.elements, selection = ids) {
    const result = duplicateSelection(scene, selection, source);
    if (change(result.scene)) ids = result.ids;
  }
  function arrange(action: string) {
    if (action === "copy") {
      clipboard = JSON.parse(JSON.stringify(selected));
      return;
    }
    if (action === "paste") {
      duplicate(
        clipboard,
        clipboard.map((e) => e.id),
      );
      return;
    }
    if (action === "duplicate") {
      duplicate();
      return;
    }
    if (action === "remove") {
      remove();
      return;
    }
    if (action === "lock") {
      patchSelected({ locked: !selected.every((e) => e.locked) }, true);
      return;
    }
    if (action === "group") {
      if (selected.length >= 2) patchSelected({ group: crypto.randomUUID() });
      return;
    }
    if (action === "ungroup") {
      patchSelected({ group: undefined });
      return;
    }
    if (action === "front" || action === "back") {
      const chosen = scene.elements.filter((e) => ids.includes(e.id));
      const rest = scene.elements.filter((e) => !ids.includes(e.id));
      change({
        ...scene,
        elements:
          action === "front" ? [...rest, ...chosen] : [...chosen, ...rest],
      });
      return;
    }
    const items = selected.filter((e) => e.kind !== "arrow" && !e.locked);
    const area = bounds(items);
    const order = [...items].sort((a, b) =>
      action === "distribute-x"
        ? bounds([a]).x - bounds([b]).x
        : bounds([a]).y - bounds([b]).y,
    );
    change({
      ...scene,
      elements: scene.elements.map((e) => {
        if (!items.includes(e)) return e;
        const b = bounds([e]);
        let dx = 0,
          dy = 0;
        if (action === "left") dx = area.x - b.x;
        if (action === "center")
          dx = area.x + area.width / 2 - b.x - b.width / 2;
        if (action === "right") dx = area.x + area.width - b.x - b.width;
        if (action === "top") dy = area.y - b.y;
        if (action === "middle")
          dy = area.y + area.height / 2 - b.y - b.height / 2;
        if (action === "bottom") dy = area.y + area.height - b.y - b.height;
        if (action === "distribute-x" && order.length >= 3)
          dx =
            area.x +
            (order.indexOf(e) * (area.width - b.width)) / (order.length - 1) -
            b.x;
        if (action === "distribute-y" && order.length >= 3)
          dy =
            area.y +
            (order.indexOf(e) * (area.height - b.height)) / (order.length - 1) -
            b.y;
        return moveObject(e, dx, dy);
      }),
    });
  }
  function zoomTo(next: number, client?: Point) {
    next = Math.max(0.15, Math.min(4, next));
    const anchor = client || { x: width / 2, y: height / 2 };
    camera = {
      x: camera.x + anchor.x / camera.zoom - anchor.x / next,
      y: camera.y + anchor.y / camera.zoom - anchor.y / next,
      zoom: next,
    };
  }
  function fit() {
    const area = bounds(scene.elements);
    const z = Math.max(
      0.15,
      Math.min(
        1.5,
        (width - 100) / (area.width + 100),
        (height - 100) / (area.height + 100),
      ),
    );
    camera = {
      x: area.x + area.width / 2 - width / z / 2,
      y: area.y + area.height / 2 - height / z / 2,
      zoom: z,
    };
  }
  function wheel(e: WheelEvent) {
    e.preventDefault();
    if (e.ctrlKey || e.metaKey) {
      const r = stage!.getBoundingClientRect();
      zoomTo(camera.zoom * Math.exp(-e.deltaY * 0.005), {
        x: e.clientX - r.left,
        y: e.clientY - r.top,
      });
    } else
      camera = {
        ...camera,
        x: camera.x + (e.shiftKey ? e.deltaY : e.deltaX) / camera.zoom,
        y: camera.y + (e.shiftKey ? 0 : e.deltaY) / camera.zoom,
      };
  }
  function down(e: PointerEvent) {
    if (!svg || gesture || (e.button !== 0 && e.button !== 1)) return;
    finishText();
    if (e.pointerType === "mouse") e.preventDefault();
    svg.focus({ preventScroll: true });
    svg.setPointerCapture(e.pointerId);
    const target = e.target as Element;
    const hit = target.closest<SVGElement>("[data-object]")?.dataset.object;
    const node = scene.elements.find((n) => n.id === hit);
    const point = location(e.clientX, e.clientY);
    const original = JSON.parse(JSON.stringify(scene)) as Scene;
    const base: Gesture = {
      type: "marquee",
      pointer: e.pointerId,
      start: point,
      client: { x: e.clientX, y: e.clientY },
      scene: original,
      ids: [...ids],
      camera: { ...camera },
    };
    if (tool === "hand" || e.button === 1 || e.altKey) {
      gesture = { ...base, type: "pan" };
      return;
    }
    if (!editable) {
      if (hit) choose(hit, e.shiftKey);
      else ids = [];
      return;
    }
    const port = target.closest<SVGElement>("[data-port]");
    const handle = target.closest<SVGElement>("[data-handle]");
    if (handle) {
      const object = scene.elements.find((n) => n.id === handle.dataset.target);
      if (object && !object.locked)
        gesture = {
          ...base,
          type: "resize",
          object,
          handle: handle.dataset.handle,
        };
      return;
    }
    if (tool === "connector") {
      const source = port?.dataset.target || hit;
      if (source && shapes.some((n) => n.id === source)) {
        if (from && from !== source && !port) {
          connect(from, source);
          return;
        }
        from = source;
        fromPort = (port?.dataset.port || "auto") as Port;
        gesture = { ...base, type: "wire", from: source, port: fromPort };
        wire = { from: bounded(point), to: bounded(point) };
      }
      return;
    }
    if (tool === "pen") {
      ids = [];
      gesture = { ...base, type: "pen", points: [bounded(point)] };
      return;
    }
    if (tool === "eraser") {
      if (hit) remove([hit], false);
      return;
    }
    if (node) {
      if (e.shiftKey || !ids.includes(node.id)) choose(node.id, e.shiftKey);
      gesture = { ...base, type: "move", ids: [...ids] };
    } else {
      if (!e.shiftKey) ids = [];
      gesture = { ...base, type: "marquee", ids: e.shiftKey ? [...ids] : [] };
    }
  }
  function move(e: PointerEvent) {
    const g = gesture;
    if (!g || g.pointer !== e.pointerId) return;
    const p = location(e.clientX, e.clientY),
      dx = snapped(p.x - g.start.x),
      dy = snapped(p.y - g.start.y);
    if (g.type === "pan")
      camera = {
        ...g.camera,
        x: g.camera.x - (e.clientX - g.client.x) / g.camera.zoom,
        y: g.camera.y - (e.clientY - g.client.y) / g.camera.zoom,
      };
    if (g.type === "move") scene = moveSelection(g.scene, g.ids, dx, dy);
    if (g.type === "marquee")
      marquee = {
        x: Math.min(p.x, g.start.x),
        y: Math.min(p.y, g.start.y),
        width: Math.abs(p.x - g.start.x),
        height: Math.abs(p.y - g.start.y),
      };
    if (g.type === "wire" && wire) wire = { ...wire, to: bounded(p) };
    if (g.type === "pen" && g.points) {
      if (g.points.length < 2000) {
        g.points.push(bounded(p));
        preview = [...g.points];
      } else
        localError =
          "Stroke limit reached. Lift your pointer and continue with another stroke.";
    }
    if (g.type === "resize" && g.object) {
      const n = g.object,
        h = g.handle || "se";
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
  function up(e: PointerEvent) {
    const g = gesture;
    if (!g || g.pointer !== e.pointerId) return;
    if (svg?.hasPointerCapture(e.pointerId))
      svg.releasePointerCapture(e.pointerId);
    if (e.type === "pointercancel") {
      scene = g.scene;
      gesture = null;
      preview = [];
      marquee = null;
      wire = null;
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
        color: defaults.color || "ink",
        strokeWidth: defaults.strokeWidth || 2,
        points: g.points,
      };
      if (
        change({ ...g.scene, elements: [...g.scene.elements, stroke] }, false)
      )
        ids = [stroke.id];
    }
    if (g.type === "marquee" && marquee) {
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
    }
    if (g.type === "wire") {
      const hit = document
        .elementFromPoint(e.clientX, e.clientY)
        ?.closest<SVGElement>("[data-object],[data-port]");
      const dest = hit?.dataset.target || hit?.dataset.object;
      if (dest && dest !== g.from && shapes.some((n) => n.id === dest))
        connect(g.from, dest, g.port, (hit?.dataset.port || "auto") as Port);
      wire = null;
    }
    if (
      g.type !== "wire" &&
      JSON.stringify(scene) !== JSON.stringify(g.scene)
    ) {
      history = [...history.slice(-49), JSON.stringify(g.scene)];
      future = [];
    }
    gesture = null;
    preview = [];
    marquee = null;
  }
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
    textEdit = "";
    if (cancel) scene = JSON.parse(textBefore);
    else if (JSON.stringify(scene) !== textBefore) {
      history = [...history.slice(-49), textBefore];
      future = [];
    }
  }
  function topmostAt(p: Point) {
    for (const e of [...scene.elements].reverse()) {
      if (e.kind === "arrow" || e.kind === "stroke") continue;
      if (
        p.x >= e.x &&
        p.x <= e.x + e.width &&
        p.y >= e.y &&
        p.y <= e.y + e.height
      )
        return e;
    }
    return undefined;
  }
  function doubleClick(e: MouseEvent) {
    // Pointer capture retargets the synthesized double-click to the SVG, so the
    // event target is not reliable here. Hit-test the scene by coordinates.
    const point = location(e.clientX, e.clientY);
    const hit = topmostAt(point);
    if (hit) void editText(hit.id);
    else if (editable) add("text", bounded(point));
  }
  function keys(e: KeyboardEvent) {
    if (document.querySelector('[aria-modal="true"]')) return;
    const target = e.target as HTMLElement;
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
      e.preventDefault();
      finishText();
      void tick().then(() => {
        if (editable && dirty) onsave(false);
      });
      return;
    }
    if (
      target.isContentEditable ||
      ["INPUT", "TEXTAREA"].includes(target.tagName) ||
      target.closest('[role="listbox"], [role="menu"]')
    )
      return;
    if (e.key === "Escape") {
      finishText(true);
      ids = [];
      from = "";
      tool = "select";
      library = matchMedia("(min-width:60rem)").matches;
      properties = false;
      return;
    }
    if (
      (e.ctrlKey || e.metaKey) &&
      ["z", "y", "a", "c", "v", "d", "g"].includes(e.key.toLowerCase())
    ) {
      e.preventDefault();
      const key = e.key.toLowerCase();
      if (key === "a") ids = scene.elements.map((n) => n.id);
      if (key === "z") e.shiftKey ? redo() : undo();
      if (key === "y") redo();
      if (key === "c") arrange("copy");
      if (key === "v") arrange("paste");
      if (key === "d") arrange("duplicate");
      if (key === "g") arrange(e.shiftKey ? "ungroup" : "group");
      return;
    }
    const steps: Record<string, [number, number]> = {
      ArrowLeft: [-1, 0],
      ArrowRight: [1, 0],
      ArrowUp: [0, -1],
      ArrowDown: [0, 1],
    };
    if (steps[e.key] && ids.length) {
      e.preventDefault();
      const [x, y] = steps[e.key];
      change(
        moveSelection(
          scene,
          ids,
          x * (e.shiftKey ? 10 : 1),
          y * (e.shiftKey ? 10 : 1),
        ),
      );
    }
    if (e.key === "Delete" || e.key === "Backspace") {
      e.preventDefault();
      remove();
    }
    if (e.key === "Enter" && object) void editText(object.id);
    if (!e.ctrlKey && !e.metaKey) {
      if (e.key.toLowerCase() === "v") tool = "select";
      if (e.key.toLowerCase() === "h") tool = "hand";
      if (e.key.toLowerCase() === "p" && editable) tool = "pen";
      if (e.key.toLowerCase() === "l" && editable) tool = "connector";
    }
  }
  function setTool(next: Tool) {
    finishText();
    if (
      next === "eraser" &&
      !confirm(
        "Erase whole objects with a tap or click? Undo can restore them in this editing session.",
      )
    )
      return;
    tool = next;
    from = "";
  }
  function drop(e: DragEvent) {
    e.preventDefault();
    const kind = e.dataTransfer?.getData(
      "application/x-trestle-shape",
    ) as ObjectKind;
    if (kinds.includes(kind) && kind !== "arrow" && kind !== "stroke")
      add(kind, bounded(location(e.clientX, e.clientY)));
  }
  function addPoints() {
    const points = strokeInput
      .trim()
      .split(/\s+/)
      .map((p) => {
        const [x, y] = p.split(",").map(Number);
        return { x, y };
      });
    const stroke: BoardObject = {
      id: crypto.randomUUID(),
      kind: "stroke",
      x: 0,
      y: 0,
      width: 0,
      height: 0,
      text: "",
      color: defaults.color || "ink",
      points,
    };
    if (change({ ...scene, elements: [...scene.elements, stroke] }))
      ids = [stroke.id];
  }
  async function fileAction(action: string) {
    if (action === "archive") {
      if (
        confirm(
          board.archived
            ? "Restore this board so Workspace members can edit it?"
            : "Archive this board? It remains readable and can be restored.",
        )
      )
        onsave(!board.archived);
      return;
    }
    finishText();
    await tick();
    if (!svg) return;
    try {
      exporting = true;
      if (action === "json")
        download(
          JSON.stringify(
            {
              format: "trestle-board",
              version: 1,
              title,
              kind: board.kind,
              scene,
            },
            null,
            2,
          ),
          "application/json",
          "trestle-board.json",
        );
      if (action === "svg")
        download(svgExport(svg), "image/svg+xml", "trestle-board.svg");
      if (action === "png") await pngExport(svg);
    } catch {
      localError =
        "Could not export this drawing. Try SVG or JSON to preserve your work.";
    } finally {
      exporting = false;
    }
  }
  function template(kind: string) {
    if (
      scene.elements.length &&
      !confirm(
        "Replace this drawing with the template? Undo can restore the current drawing before you leave.",
      )
    )
      return;
    const start = {
      ...make("terminator"),
      x: 160,
      y: 160,
      text: "Start",
      fill: "success" as Ink,
      color: "success" as Ink,
    };
    const process = { ...make("rectangle"), x: 520, y: 160, text: "Process" };
    const decision = {
      ...make("diamond"),
      x: 880,
      y: 130,
      text: "Decision",
      fill: "warning" as Ink,
      color: "warning" as Ink,
    };
    const end = {
      ...make("terminator"),
      x: 880,
      y: 500,
      text: "End",
      fill: "success" as Ink,
      color: "success" as Ink,
    };
    const edge = (
      a: BoardObject,
      b: BoardObject,
      text = "",
      fromPort: Port = "auto",
      toPort: Port = "auto",
    ): BoardObject => ({
      id: crypto.randomUUID(),
      kind: "arrow",
      x: 0,
      y: 0,
      width: 0,
      height: 0,
      text,
      color: "ink",
      route: "elbow",
      from: a.id,
      to: b.id,
      fromPort,
      toPort,
    });
    const elements =
      kind === "swimlane"
        ? [
            {
              ...make("swimlane"),
              x: 80,
              y: 60,
              width: 1240,
              height: 740,
              text: "Lane 1 / Lane 2 / Lane 3",
            },
            start,
            process,
            decision,
            end,
            edge(start, process),
            edge(process, decision),
            edge(decision, end, "Yes"),
            edge(decision, process, "No", "top", "top"),
          ]
        : [
            start,
            process,
            decision,
            end,
            edge(start, process),
            edge(process, decision),
            edge(decision, end, "Yes"),
            edge(decision, process, "No", "top", "top"),
          ];
    if (change({ version: 1, elements })) {
      templates = false;
      ids = [];
      void tick().then(fit);
    }
  }
  onMount(() => {
    const narrow = matchMedia("(max-width:60rem)");
    library = !narrow.matches;
    let fitted = false;
    const observer = new ResizeObserver((entries) => {
      const rect = entries[0].contentRect;
      width = rect.width;
      height = rect.height;
      if (!fitted && width > 0 && height > 0) {
        fitted = true;
        fit();
      }
    });
    if (stage) observer.observe(stage);
    const resetPanels = (event: MediaQueryListEvent) => {
      if (event.matches) {
        library = false;
        properties = false;
      }
    };
    narrow.addEventListener("change", resetPanels);
    return () => {
      observer.disconnect();
      narrow.removeEventListener("change", resetPanels);
    };
  });
</script>

<svelte:window onkeydown={keys} />
<section
  class="board-editor"
  aria-label={board.kind === "diagram" ? "Diagram editor" : "Whiteboard editor"}
>
  <h1 class="visually-hidden">
    {title} — {board.kind === "diagram" ? "Diagram" : "Whiteboard"}
  </h1>
  <header class="editor-head">
    <button
      type="button"
      class="btn-quiet btn-icon"
      aria-label="Back to boards"
      onclick={onback}
      disabled={saving}><Icon name="arrow-left" size={20} /></button
    >
    <span class="editor-mark tile"
      ><Icon
        name={board.kind === "diagram" ? "workflow" : "pencil"}
        size={20}
      /></span
    >
    <div class="document-heading">
      <label for="board-title" class="visually-hidden">Board title</label><input
        id="board-title"
        bind:value={title}
        maxlength="120"
        disabled={!editable}
      />
      <p>
        {workspaceName}<span aria-hidden="true"> / </span><span
          >{board.kind === "diagram" ? "Diagram" : "Whiteboard"}</span
        >
      </p>
    </div>
    <span class="save-state" role="status"
      ><Icon
        name={saving ? "clock" : dirty ? "circle" : "check"}
      />{board.archived
        ? "Archived"
        : saving
          ? "Saving…"
          : dirty
            ? "Unsaved changes"
            : "All changes saved"}</span
    >
    <Select
      label="Export board"
      placeholder="Export"
      value=""
      variant="ghost"
      disabled={exporting}
      options={[
        { value: "json", label: "Export JSON draft", icon: "download" },
        { value: "svg", label: "Export SVG", icon: "image" },
        { value: "png", label: "Export PNG", icon: "image" },
        ...(canArchive && board.id
          ? [
              {
                value: "archive",
                label: board.archived ? "Restore board" : "Archive board",
                icon: "archive" as IconName,
                disabled: saving || dirty,
              },
            ]
          : []),
      ]}
      onchange={fileAction}
    />
    {#if canEdit && !board.archived}<Button
        variant="primary"
        disabled={saving || !dirty || !!gesture || exporting}
        onclick={() => {
          finishText();
          void tick().then(() => onsave(false));
        }}>Save board</Button
      >{/if}
  </header>
  <div class="editor-toolbar" aria-label="Drawing and formatting tools">
    <div class="tool-group">
      <button
        type="button"
        class="btn-quiet btn-icon"
        aria-label="Shapes and layers"
        aria-expanded={library}
        onclick={() => {
          library = !library;
          if (library && matchMedia("(max-width: 60rem)").matches)
            properties = false;
        }}><Icon name="layers" size={20} /></button
      ><button
        type="button"
        class="btn-quiet btn-icon"
        aria-label="Undo"
        disabled={!editable || !history.length}
        onclick={undo}><Icon name="undo" size={20} /></button
      ><button
        type="button"
        class="btn-quiet btn-icon"
        aria-label="Redo"
        disabled={!editable || !future.length}
        onclick={redo}><Icon name="redo" size={20} /></button
      >
    </div>
    <div class="tool-group">
      {#each [{ value: "select", label: "Select objects", icon: "select" }, { value: "hand", label: "Pan canvas", icon: "hand" }, { value: "connector", label: "Connect shapes", icon: "connector" }, { value: "pen", label: "Draw freehand", icon: "pencil" }, { value: "eraser", label: "Erase object", icon: "trash" }] as t}<button
          type="button"
          class="btn-quiet btn-icon"
          class:active-tool={tool === t.value}
          aria-label={t.label}
          aria-pressed={tool === t.value}
          disabled={!editable && !["hand", "select"].includes(t.value)}
          onclick={() => setTool(t.value as Tool)}
          ><Icon name={t.icon as IconName} size={20} /></button
        >{/each}
    </div>
    <div class="tool-group">
      <Select
        label="Fill color"
        value={format.fill || "none"}
        variant="ghost"
        disabled={!editable || !!object?.locked}
        options={[
          { value: "none", label: "No fill" },
          ...colors.map((c) => ({
            value: c,
            label: `${c} fill`,
            color: `var(${c === "ink" ? "--text" : `--${c}`})`,
          })),
        ]}
        onchange={(v) => patchSelected({ fill: v as Ink | "none" })}
      /><Select
        label="Stroke color"
        value={format.color || "ink"}
        variant="ghost"
        disabled={!editable || !!object?.locked}
        options={colors.map((c) => ({
          value: c,
          label: `${c} line`,
          color: `var(${c === "ink" ? "--text" : `--${c}`})`,
        }))}
        onchange={(v) => patchSelected({ color: v as Ink })}
      /><Select
        label="Line width"
        value={String(format.strokeWidth || 2)}
        variant="ghost"
        disabled={!editable}
        options={[1, 2, 3, 4, 6, 8].map((n) => ({
          value: String(n),
          label: `${n} px`,
        }))}
        onchange={(v) => patchSelected({ strokeWidth: Number(v) })}
      />
    </div>
    <div class="tool-group">
      <Select
        label="Font size"
        value={String(format.fontSize || 18)}
        variant="ghost"
        disabled={!editable}
        options={[12, 14, 18, 24, 32, 48, 72].map((n) => ({
          value: String(n),
          label: `${n} pt`,
        }))}
        onchange={(v) => patchSelected({ fontSize: Number(v) })}
      /><button
        type="button"
        class="btn-quiet bold-button"
        aria-label="Bold text"
        aria-pressed={!!format.bold}
        disabled={!editable}
        onclick={() => patchSelected({ bold: !format.bold })}>B</button
      ><Select
        label="Text alignment"
        value={format.align || "center"}
        variant="ghost"
        disabled={!editable}
        options={[
          { value: "left", label: "Left text" },
          { value: "center", label: "Center text" },
          { value: "right", label: "Right text" },
        ]}
        onchange={(v) => patchSelected({ align: v as BoardObject["align"] })}
      />
    </div>
    <div class="tool-group">
      <Select
        label="Connector routing"
        value={format.route || "elbow"}
        variant="ghost"
        disabled={!editable}
        options={[
          { value: "elbow", label: "Elbow line" },
          { value: "straight", label: "Straight line" },
          { value: "curve", label: "Curved line" },
        ]}
        onchange={(v) => {
          defaults = { ...defaults, route: v as BoardObject["route"] };
          if (object?.kind === "arrow")
            patchSelected({ route: v as BoardObject["route"] });
        }}
      /><Select
        label="Arrange objects"
        placeholder="Arrange"
        value=""
        variant="ghost"
        disabled={!editable}
        options={[
          { value: "copy", label: "Copy objects", disabled: !ids.length },
          {
            value: "paste",
            label: "Paste objects",
            disabled: !clipboard.length,
          },
          {
            value: "duplicate",
            label: "Duplicate objects",
            disabled: !ids.length,
          },
          { value: "front", label: "Bring to front", disabled: !ids.length },
          { value: "back", label: "Send to back", disabled: !ids.length },
          { value: "group", label: "Group objects", disabled: ids.length < 2 },
          {
            value: "ungroup",
            label: "Ungroup objects",
            disabled: !object?.group,
          },
          {
            value: "lock",
            label: object?.locked ? "Unlock objects" : "Lock objects",
            disabled: !ids.length,
          },
          ...[
            ["left", "Align left"],
            ["center", "Align center"],
            ["right", "Align right"],
            ["top", "Align top"],
            ["middle", "Align middle"],
            ["bottom", "Align bottom"],
            ["distribute-x", "Distribute horizontally"],
            ["distribute-y", "Distribute vertically"],
          ].map(([value, label]) => ({
            value,
            label,
            disabled: ids.length < (value.startsWith("distribute") ? 3 : 2),
          })),
          { value: "remove", label: "Remove objects", disabled: !ids.length },
        ]}
        onchange={arrange}
      />
    </div>
    <button
      type="button"
      class="btn-quiet btn-icon"
      aria-label="Object properties"
      aria-expanded={properties}
      onclick={() => {
        properties = !properties;
        if (properties && matchMedia("(max-width: 60rem)").matches)
          library = false;
      }}><Icon name="settings" size={20} /></button
    >
  </div>
  {#if error || localError}<div class="editor-error">
      <Feedback tone="danger">{error || localError}</Feedback
      >{#if error && board.id}<Button disabled={saving} onclick={onreload}
          >Reload saved board</Button
        >{/if}
    </div>{/if}
  {#if !canEdit || board.archived}<Feedback
      >{board.archived
        ? "Archived board. Owners and admins can restore it from Export."
        : "Read-only board. You can inspect, pan, zoom and export."}</Feedback
    >{/if}
  <div class="editor-body">
    {#if library}<aside class="shape-library" aria-label="Shape library">
        <div class="library-tabs">
          <button
            type="button"
            class="btn-quiet"
            aria-pressed={libraryTab === "shapes"}
            onclick={() => (libraryTab = "shapes")}>Shapes</button
          ><button
            type="button"
            class="btn-quiet"
            aria-pressed={libraryTab === "objects"}
            onclick={() => (libraryTab = "objects")}>Objects</button
          ><button
            type="button"
            class="btn-quiet btn-icon close-library"
            aria-label="Close shape library"
            onclick={() => (library = false)}><Icon name="close" /></button
          >
        </div>
        {#if libraryTab === "shapes"}<div class="shape-search">
            <Icon name="search" /><label
              for="shape-search"
              class="visually-hidden">Search shapes</label
            ><input
              id="shape-search"
              type="search"
              placeholder="Search shapes"
              bind:value={query}
            />
          </div>
          {#each ["Standard", "Flowchart", "Containers"] as section}{@const entries =
              catalog.filter(
                (s) =>
                  s.section === section &&
                  `${s.kind} ${s.label}`
                    .toLowerCase()
                    .includes(query.toLowerCase()),
              )}{#if entries.length}<details open class="shape-section">
                <summary>{section}</summary>
                <div class="shape-palette">
                  {#each entries as entry}<button
                      type="button"
                      class="btn-quiet shape-button"
                      aria-label={`Add ${entry.kind}`}
                      title={entry.label}
                      disabled={!editable}
                      draggable={editable}
                      ondragstart={(e) => {
                        e.dataTransfer?.setData(
                          "application/x-trestle-shape",
                          entry.kind,
                        );
                        if (e.dataTransfer)
                          e.dataTransfer.effectAllowed = "copy";
                      }}
                      onclick={() => add(entry.kind)}
                      ><Icon name={entry.icon} size={27} /><span
                        >{entry.label}</span
                      ></button
                    >{/each}
                </div>
              </details>{/if}{/each}{#if !catalog.some( (s) => `${s.kind} ${s.label}`
                .toLowerCase()
                .includes(query.toLowerCase()) )}<p class="library-note">
              No matching shapes. Try process, decision or text.
            </p>{/if}
          <div class="library-bottom">
            <Button disabled={!editable} onclick={() => (templates = true)}
              ><Icon name="workflow" />Start from template</Button
            >
            <p class="note">
              Click to add. Drag to place. Double-click a shape to write.
            </p>
          </div>
        {:else}<div class="object-list">
            {#each [...scene.elements].reverse() as e, i (e.id)}<button
                type="button"
                class="btn-quiet layer-row"
                aria-pressed={ids.includes(e.id)}
                onclick={(event) => choose(e.id, event.shiftKey)}
                ><Icon
                  name={e.locked
                    ? "lock"
                    : catalog.find((s) => s.kind === e.kind)?.icon ||
                      "connector"}
                /><span
                  >{e.text || `${e.kind} ${scene.elements.length - i}`}</span
                ></button
              >{/each}{#if !scene.elements.length}<p class="library-note">
                Your shapes and lines will appear here.
              </p>{/if}
          </div>{/if}
      </aside>{/if}
    <!-- svelte-ignore a11y_no_static_element_interactions (Native shape drag/drop is an optional alternative to palette buttons.) -->
    <div
      class="editor-stage"
      class:panning={tool === "hand"}
      class:drawing={tool === "pen" || tool === "connector"}
      bind:this={stage}
      ondragover={(e) => e.preventDefault()}
      ondrop={drop}
    >
      <BoardCanvas
        {scene}
        preview={{
          points: preview,
          color: defaults.color || "ink",
          width: defaults.strokeWidth || 2,
        }}
        selectedIds={ids}
        bind:svg
        viewBox={`${camera.x} ${camera.y} ${width / camera.zoom} ${height / camera.zoom}`}
        scale={camera.zoom}
        {grid}
        connecting={tool === "connector" && editable}
        {marquee}
        {wire}
        onpointerdown={down}
        onpointermove={move}
        onpointerup={up}
        ondblclick={doubleClick}
        onwheel={wheel}
      />
      {#if editedObject}<textarea
          class="canvas-text-editor"
          aria-label="Edit shape text"
          bind:this={labelInput}
          value={textDraft}
          oninput={(e) => {
            textDraft = e.currentTarget.value;
            change(
              {
                ...scene,
                elements: scene.elements.map((n) =>
                  n.id === textEdit ? { ...n, text: textDraft } : n,
                ),
              },
              false,
            );
          }}
          maxlength="200"
          style:--label-left={`${(editedObject.x - camera.x) * camera.zoom}px`}
          style:--label-top={`${(editedObject.y - camera.y) * camera.zoom}px`}
          style:--label-width={`${Math.max(140, editedObject.width * camera.zoom)}px`}
          style:--label-height={`${Math.max(60, editedObject.height * camera.zoom)}px`}
          onblur={() => finishText()}
          onkeydown={(e) => {
            if (e.key === "Escape") {
              e.stopPropagation();
              finishText(true);
            }
            if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
              e.preventDefault();
              finishText();
              svg?.focus();
            }
          }}></textarea>{/if}
      {#if !scene.elements.length}<div
          class="canvas-invitation"
          aria-hidden="true"
        >
          <Icon
            name={board.kind === "diagram" ? "workflow" : "pencil"}
            size={28}
          /><span
            >{board.kind === "diagram"
              ? "Make the process visible"
              : "A place to think visually"}</span
          >
          <p>Add a shape, start drawing or choose a template.</p>
        </div>{/if}
    </div>
    {#if properties}<aside
        class="properties-panel"
        aria-label="Object inspector"
      >
        <div class="properties-head">
          <h3>Properties</h3>
          <button
            type="button"
            class="btn-quiet btn-icon"
            aria-label="Close properties"
            onclick={() => (properties = false)}><Icon name="close" /></button
          >
        </div>
        <Select
          label="Selected object"
          value={object?.id || ""}
          options={[
            { value: "", label: "Choose an object" },
            ...scene.elements.map((e, i) => ({
              value: e.id,
              label: `${i + 1}. ${e.kind}${e.text ? `: ${e.text}` : ""}`,
            })),
          ]}
          onchange={(v) => (ids = v ? [v] : [])}
        />
        {#if object}<div class="field">
            <label for="object-text">Object label</label><textarea
              id="object-text"
              value={object.text}
              maxlength="200"
              disabled={!editable || object.locked}
              oninput={(e) => patchSelected({ text: e.currentTarget.value })}
            ></textarea>
          </div>
          {#if object.kind !== "arrow" && object.kind !== "stroke"}<div
              class="geometry-fields"
            >
              {#each ["x", "y", "width", "height"] as key}<div class="field">
                  <label for={`object-${key}`}
                    >{key === "x"
                      ? "Position X"
                      : key === "y"
                        ? "Position Y"
                        : key === "width"
                          ? "Width"
                          : "Height"}</label
                  ><input
                    id={`object-${key}`}
                    type="number"
                    min={key === "width" || key === "height" ? 20 : 0}
                    max="4000"
                    value={object[key as "x"]}
                    disabled={!editable || object.locked}
                    onchange={(e) =>
                      patchSelected({ [key]: e.currentTarget.valueAsNumber })}
                  />
                </div>{/each}
            </div>{/if}<Select
            label="Line style"
            value={object.dashed ? "dashed" : "solid"}
            options={[
              { value: "solid", label: "Solid line" },
              { value: "dashed", label: "Dashed line" },
            ]}
            onchange={(v) => patchSelected({ dashed: v === "dashed" })}
            disabled={!editable}
          />{#if object.kind === "arrow"}<Select
              label="Arrowheads"
              value={object.heads || "end"}
              options={[
                { value: "end", label: "End arrow" },
                { value: "both", label: "Both ends" },
                { value: "none", label: "No arrows" },
              ]}
              onchange={(v) =>
                patchSelected({ heads: v as BoardObject["heads"] })}
              disabled={!editable}
            />{/if}{#if editable}<Button
              variant="danger-quiet"
              onclick={() => remove()}>Remove object</Button
            >{/if}{/if}
        <h3>Connections</h3>
        <div class="field">
          <label for="arrow-from">From shape</label><Select
            id="arrow-from"
            label="Arrow from"
            value={from}
            options={shapes.map((e, i) => ({
              value: e.id,
              label: e.text || `${e.kind} ${i + 1}`,
            }))}
            onchange={(v) => (from = v)}
            disabled={!editable}
          />
        </div>
        <div class="field">
          <label for="arrow-to">To shape</label><Select
            id="arrow-to"
            label="Arrow to"
            value={to}
            options={shapes.map((e, i) => ({
              value: e.id,
              label: e.text || `${e.kind} ${i + 1}`,
            }))}
            onchange={(v) => (to = v)}
            disabled={!editable}
          />
        </div>
        <Button
          disabled={!editable || !from || !to || from === to}
          onclick={() => connect()}>Connect with arrow</Button
        >
        <details class="precision-drawing">
          <summary>Draw with coordinates</summary>
          <div class="field">
            <label for="stroke-points">Stroke points</label><textarea
              id="stroke-points"
              bind:value={strokeInput}
              placeholder="100,100 150,130 200,100"
              disabled={!editable}></textarea>
            <p class="note">Space-separated x,y pairs, from 0 to 4000.</p>
          </div>
          <Button
            disabled={!editable || !strokeInput.trim()}
            onclick={addPoints}>Add stroke from points</Button
          >
        </details>
      </aside>{/if}
  </div>
  <footer class="editor-foot">
    <p id="board-canvas-help">{hint}</p>
    <span class="object-count num">{scene.elements.length} objects</span><button
      type="button"
      class="btn-quiet"
      aria-pressed={grid}
      onclick={() => (grid = !grid)}>Grid</button
    ><button
      type="button"
      class="btn-quiet"
      aria-pressed={snap}
      onclick={() => (snap = !snap)}>Snap</button
    ><button
      type="button"
      class="btn-quiet btn-icon"
      aria-label="Zoom out"
      onclick={() => zoomTo(camera.zoom / 1.2)}><Icon name="zoom-out" /></button
    ><Select
      label="Canvas zoom"
      value={String(Math.round(camera.zoom * 100))}
      placeholder={`${Math.round(camera.zoom * 100)}%`}
      variant="ghost"
      options={[25, 50, 75, 100, 125, 150, 200, 300, 400].map((n) => ({
        value: String(n),
        label: `${n}%`,
      }))}
      onchange={(v) => zoomTo(Number(v) / 100)}
    /><button
      type="button"
      class="btn-quiet btn-icon"
      aria-label="Zoom in"
      onclick={() => zoomTo(camera.zoom * 1.2)}><Icon name="zoom-in" /></button
    ><button
      type="button"
      class="btn-quiet btn-icon"
      aria-label="Fit drawing"
      onclick={fit}><Icon name="fit" /></button
    >
  </footer>
</section>
{#if templates}<Modal
    title="Start from a template"
    description="Editable starting points, not example business records. Save the result as your own board."
    onclose={() => (templates = false)}
    ><div class="template-options">
      <Button onclick={() => template("flowchart")}
        ><Icon name="workflow" size={24} />Basic flowchart</Button
      ><Button onclick={() => template("swimlane")}
        ><Icon name="swimlane" size={24} />Three-lane process</Button
      >
    </div></Modal
  >{/if}

<style>
  .board-editor {
    display: flex;
    flex-direction: column;
    height: 100dvh;
    min-height: 0;
    background: var(--surface);
    color: var(--text);
  }
  .editor-head {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-4);
    border-bottom: 1px solid var(--line);
    flex: none;
    min-width: 0;
  }
  .editor-mark {
    flex: none;
  }
  .document-heading {
    min-width: 0;
    flex: 1;
  }
  .document-heading input {
    border: 1px solid transparent;
    background: transparent;
    font-size: var(--text-section);
    font-weight: 540;
    padding: var(--space-1);
    min-height: var(--control-height);
    width: min(100%, 32rem);
    color: var(--text);
  }
  .document-heading input:hover {
    border-color: var(--border);
  }
  .document-heading p {
    font-size: var(--text-xs);
    padding-inline: var(--space-1);
  }
  .save-state {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    color: var(--text-muted);
    font-size: var(--text-small);
    white-space: nowrap;
  }
  .editor-toolbar {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    padding: var(--space-1) var(--space-3);
    gap: var(--space-1);
    border-bottom: 1px solid var(--line);
    background: var(--surface-raised);
    flex: none;
  }
  .tool-group {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    padding-inline: var(--space-2);
    border-right: 1px solid var(--line);
  }
  .active-tool,
  button[aria-pressed="true"] {
    background: var(--accent-soft);
    color: var(--accent);
  }
  .bold-button {
    min-width: var(--control-height);
    font-weight: 600;
    font-size: var(--text-lead);
  }
  .editor-body {
    display: flex;
    flex: 1;
    min-height: 0;
    position: relative;
    min-width: 0;
    overflow: hidden;
  }
  .shape-library {
    display: flex;
    flex-direction: column;
    width: 14.5rem;
    flex: none;
    border-right: 1px solid var(--line);
    background: var(--surface-raised);
    overflow: auto;
  }
  .library-tabs {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-2);
    border-bottom: 1px solid var(--line);
    position: sticky;
    top: 0;
    z-index: var(--layer-sticky);
    background: var(--surface-raised);
  }
  .library-tabs > button {
    flex: 1;
    min-width: 0;
    padding-inline: var(--space-2);
    white-space: nowrap;
  }
  .library-tabs > .close-library {
    flex: none;
    padding-inline: var(--space-1);
  }
  .shape-search {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3);
    position: sticky;
    top: calc(var(--control-height) + var(--space-4));
    z-index: var(--layer-sticky);
    background: var(--surface-raised);
  }
  .shape-search input {
    min-width: 0;
    width: 100%;
  }
  .shape-section {
    border-bottom: 1px solid var(--line);
    padding: var(--space-3);
  }
  summary {
    cursor: pointer;
    color: var(--text-secondary);
    font-weight: 540;
    min-height: var(--control-height);
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }
  summary::before {
    content: "›";
    font-size: var(--text-section);
  }
  details[open] > summary::before {
    content: "⌄";
  }
  .shape-palette {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--space-1);
  }
  .shape-button {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    align-items: center;
    justify-content: center;
    min-height: 4.5rem;
    padding: var(--space-2) var(--space-1);
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
  }
  .shape-button:hover {
    border-color: var(--border);
    background: var(--hover);
  }
  .shape-button span {
    font-size: var(--text-xs);
    line-height: var(--leading-tight);
    white-space: normal;
    text-align: center;
    overflow-wrap: normal;
  }
  .library-bottom {
    margin-top: auto;
    padding: var(--space-4) var(--space-3);
    display: grid;
    gap: var(--space-3);
  }
  .library-note {
    padding: var(--space-4);
    font-size: var(--text-small);
  }
  .object-list {
    padding: var(--space-2);
    display: grid;
    gap: var(--space-1);
  }
  .layer-row {
    display: flex;
    gap: var(--space-2);
    min-width: 0;
    text-align: left;
  }
  .layer-row span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .editor-stage {
    position: relative;
    flex: 1;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    background: var(--surface-sunken);
  }
  .editor-stage.panning {
    cursor: grab;
  }
  .editor-stage.panning:active {
    cursor: grabbing;
  }
  .editor-stage.drawing {
    cursor: crosshair;
  }
  .canvas-invitation {
    position: absolute;
    left: 50%;
    top: 44%;
    transform: translate(-50%, -50%);
    display: grid;
    justify-items: center;
    gap: var(--space-3);
    max-width: 24rem;
    width: 80%;
    text-align: center;
    pointer-events: none;
    color: var(--text-muted);
  }
  .canvas-invitation span {
    font-size: var(--text-lead);
    color: var(--text-secondary);
    font-weight: 500;
  }
  .canvas-invitation p {
    font-size: var(--text-small);
  }
  .properties-panel {
    width: 16rem;
    flex: none;
    border-left: 1px solid var(--line);
    background: var(--surface-raised);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    overflow: auto;
  }
  .properties-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .geometry-fields {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--space-3);
  }
  .precision-drawing {
    border-top: 1px solid var(--line);
  }
  .precision-drawing .field {
    margin-block: var(--space-3);
  }
  .editor-foot {
    flex: none;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-height: 3rem;
    padding: var(--space-1) var(--space-3);
    border-top: 1px solid var(--line);
    background: var(--surface-raised);
  }
  .editor-foot p {
    flex: 1;
    min-width: 0;
    font-size: var(--text-xs);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .editor-foot > button,
  .editor-foot > :global(button) {
    flex: none;
    white-space: nowrap;
  }
  .object-count {
    color: var(--text-muted);
    font-size: var(--text-xs);
    white-space: nowrap;
  }
  .editor-error {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-4);
    background: var(--danger-soft);
  }
  .canvas-text-editor {
    position: absolute;
    left: var(--label-left);
    top: var(--label-top);
    width: var(--label-width);
    height: var(--label-height);
    max-width: 100%;
    min-height: var(--control-height);
    border: 2px solid var(--accent);
    background: var(--surface-float);
    padding: var(--space-3);
    resize: none;
    z-index: var(--layer-sticky);
    font-size: var(--text-body);
  }
  .template-options {
    display: grid;
    gap: var(--space-3);
    padding-block: var(--space-3);
  }
  .template-options :global(button) {
    justify-content: flex-start;
    min-height: 4rem;
    gap: var(--space-3);
  }
  @media (max-width: 60rem) {
    .shape-library,
    .properties-panel {
      position: absolute;
      inset: 0 auto 0 0;
      width: min(17rem, 85%);
      z-index: var(--layer-sticky);
      box-shadow: var(--shadow-float);
    }
    .properties-panel {
      left: auto;
      right: 0;
    }
    .editor-toolbar {
      flex-wrap: nowrap;
      overflow: auto;
      min-height: 3.25rem;
    }
    .tool-group {
      flex: none;
    }
    .save-state {
      order: 3;
      flex: 1 0 100%;
      font-size: var(--text-xs);
    }
    .editor-mark {
      display: none;
    }
    .editor-head {
      flex-wrap: wrap;
      gap: var(--space-2);
      padding: var(--space-2);
    }
    .document-heading {
      flex-basis: 40%;
    }
    .editor-foot {
      overflow: auto;
    }
    .editor-foot p {
      display: none;
    }
    .object-count {
      margin-right: auto;
    }
    .editor-head :global(.btn-primary) {
      min-width: 5rem;
    }
  }
  @media (max-width: 25rem) {
    .document-heading {
      flex-basis: 60%;
    }
    .editor-head {
      gap: var(--space-1);
    }
    .editor-foot {
      gap: 0;
    }
    .editor-foot > button {
      padding-inline: var(--space-2);
    }
  }
</style>

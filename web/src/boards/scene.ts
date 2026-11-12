export type ObjectKind =
  | "rectangle"
  | "ellipse"
  | "diamond"
  | "text"
  | "stroke"
  | "arrow"
  | "terminator"
  | "document"
  | "database"
  | "parallelogram"
  | "note"
  | "frame"
  | "swimlane";
export type Ink = "ink" | "accent" | "success" | "warning" | "danger" | "info";
export type Port = "auto" | "top" | "right" | "bottom" | "left";
export interface Point {
  x: number;
  y: number;
}
export interface BoardObject {
  id: string;
  kind: ObjectKind;
  x: number;
  y: number;
  width: number;
  height: number;
  text: string;
  color: Ink;
  from?: string;
  to?: string;
  points?: Point[];
  fill?: Ink | "none";
  strokeWidth?: number;
  dashed?: boolean;
  fontSize?: number;
  bold?: boolean;
  align?: "left" | "center" | "right";
  route?: "straight" | "elbow" | "curve";
  fromPort?: Port;
  toPort?: Port;
  heads?: "end" | "both" | "none";
  group?: string;
  locked?: boolean;
}
export interface Scene {
  version: 1;
  elements: BoardObject[];
}
export interface Board {
  id: string;
  workspace_id: string;
  title: string;
  kind: "diagram" | "whiteboard";
  version: number;
  archived: boolean;
  updated_at: string;
  scene?: Scene;
}
export const colors = [
  "ink",
  "accent",
  "success",
  "warning",
  "danger",
  "info",
] as const;
export const kinds = [
  "rectangle",
  "ellipse",
  "diamond",
  "text",
  "stroke",
  "arrow",
  "terminator",
  "document",
  "database",
  "parallelogram",
  "note",
  "frame",
  "swimlane",
] as const;
export const inkToken: Record<Ink, string> = {
  ink: "--text",
  accent: "--accent",
  success: "--success",
  warning: "--warning",
  danger: "--danger",
  info: "--info",
};

const coordinate = (n: unknown): n is number =>
  typeof n === "number" && Number.isFinite(n) && n >= 0 && n <= 4000;
export function validateScene(value: unknown): value is Scene {
  if (!value || typeof value !== "object") return false;
  const scene = value as Scene;
  if (
    scene.version !== 1 ||
    !Array.isArray(scene.elements) ||
    scene.elements.length > 500
  )
    return false;
  const ids = new Map<string, ObjectKind>();
  let points = 0;
  for (const e of scene.elements) {
    if (
      !e ||
      typeof e !== "object" ||
      typeof e.id !== "string" ||
      !e.id ||
      new TextEncoder().encode(e.id).length > 64 ||
      ids.has(e.id) ||
      !kinds.includes(e.kind) ||
      !colors.includes(e.color) ||
      typeof e.text !== "string" ||
      [...e.text].length > 200
    )
      return false;
    if (
      ![e.x, e.y, e.width, e.height].every(coordinate) ||
      e.x + e.width > 4000 ||
      e.y + e.height > 4000
    )
      return false;
    if (
      e.kind !== "stroke" &&
      e.kind !== "arrow" &&
      (e.width < 20 || e.height < 20)
    )
      return false;
    if (e.points !== undefined && !Array.isArray(e.points)) return false;
    if (
      e.kind === "stroke"
        ? !e.points || e.points.length < 2
        : !!e.points?.length
    )
      return false;
    if (e.kind !== "arrow" && (e.from || e.to)) return false;
    if (e.fill !== undefined && e.fill !== "none" && !colors.includes(e.fill))
      return false;
    if (
      e.strokeWidth !== undefined &&
      (!Number.isFinite(e.strokeWidth) ||
        e.strokeWidth < 1 ||
        e.strokeWidth > 8)
    )
      return false;
    if (
      e.fontSize !== undefined &&
      (!Number.isFinite(e.fontSize) || e.fontSize < 12 || e.fontSize > 72)
    )
      return false;
    for (const key of ["bold", "dashed", "locked"] as const)
      if (e[key] !== undefined && typeof e[key] !== "boolean") return false;
    if (e.align !== undefined && !["left", "center", "right"].includes(e.align))
      return false;
    if (
      e.route !== undefined &&
      !["straight", "elbow", "curve"].includes(e.route)
    )
      return false;
    for (const port of [e.fromPort, e.toPort])
      if (
        port !== undefined &&
        !["auto", "top", "right", "bottom", "left"].includes(port)
      )
        return false;
    if (e.heads !== undefined && !["end", "both", "none"].includes(e.heads))
      return false;
    if (
      e.group !== undefined &&
      (typeof e.group !== "string" ||
        new TextEncoder().encode(e.group).length > 64)
    )
      return false;
    for (const p of e.points || [])
      if (!p || !coordinate(p.x) || !coordinate(p.y)) return false;
    points += e.points?.length || 0;
    ids.set(e.id, e.kind);
  }
  return (
    points <= 20000 &&
    scene.elements.every(
      (e) =>
        e.kind !== "arrow" ||
        (e.from !== e.to &&
          !!ids.get(e.from || "") &&
          !!ids.get(e.to || "") &&
          !["arrow", "stroke"].includes(ids.get(e.from!)!) &&
          !["arrow", "stroke"].includes(ids.get(e.to!)!)),
    )
  );
}

export function removeObject(scene: Scene, id: string): Scene {
  return {
    ...scene,
    elements: scene.elements.filter(
      (e) => e.id !== id && e.from !== id && e.to !== id,
    ),
  };
}

export function moveObject(
  e: BoardObject,
  dx: number,
  dy: number,
): BoardObject {
  if (e.kind === "arrow") return e;
  const xs =
    e.kind === "stroke" ? e.points!.map((p) => p.x) : [e.x, e.x + e.width];
  const ys =
    e.kind === "stroke" ? e.points!.map((p) => p.y) : [e.y, e.y + e.height];
  dx = Math.max(-Math.min(...xs), Math.min(dx, 4000 - Math.max(...xs)));
  dy = Math.max(-Math.min(...ys), Math.min(dy, 4000 - Math.max(...ys)));
  return e.kind === "stroke"
    ? { ...e, points: e.points!.map((p) => ({ x: p.x + dx, y: p.y + dy })) }
    : { ...e, x: e.x + dx, y: e.y + dy };
}

export function arrowEnds(e: BoardObject, scene: Scene) {
  const from = scene.elements.find((n) => n.id === e.from)!;
  const to = scene.elements.find((n) => n.id === e.to)!;
  // Connect at the bounding edges, so labels remain unobscured.
  const edge = (a: BoardObject, b: BoardObject, port: Port = "auto") => {
    if (port === "top") return { x: a.x + a.width / 2, y: a.y };
    if (port === "bottom") return { x: a.x + a.width / 2, y: a.y + a.height };
    if (port === "left") return { x: a.x, y: a.y + a.height / 2 };
    if (port === "right") return { x: a.x + a.width, y: a.y + a.height / 2 };
    const dx = b.x + b.width / 2 - a.x - a.width / 2;
    const dy = b.y + b.height / 2 - a.y - a.height / 2;
    let ratio = Math.max(
      Math.abs(dx) / (a.width / 2),
      Math.abs(dy) / (a.height / 2),
    );
    if (a.kind === "ellipse")
      ratio = Math.hypot(dx / (a.width / 2), dy / (a.height / 2));
    if (a.kind === "diamond")
      ratio = Math.abs(dx) / (a.width / 2) + Math.abs(dy) / (a.height / 2);
    return {
      x: a.x + a.width / 2 + (ratio ? dx / ratio : 0),
      y: a.y + a.height / 2 + (ratio ? dy / ratio : 0),
    };
  };
  return { from: edge(from, to, e.fromPort), to: edge(to, from, e.toPort) };
}

export function connectorPath(e: BoardObject, scene: Scene) {
  const { from: a, to: b } = arrowEnds(e, scene);
  const middle = (a.x + b.x) / 2;
  if (e.route === "elbow") return `M${a.x},${a.y} H${middle} V${b.y} H${b.x}`;
  if (e.route === "curve")
    return `M${a.x},${a.y} C${middle},${a.y} ${middle},${b.y} ${b.x},${b.y}`;
  return `M${a.x},${a.y} L${b.x},${b.y}`;
}

export function bounds(objects: BoardObject[]) {
  const points = objects.flatMap((e) =>
    e.kind === "arrow"
      ? []
      : e.kind === "stroke"
        ? e.points || []
        : [
            { x: e.x, y: e.y },
            { x: e.x + e.width, y: e.y + e.height },
          ],
  );
  if (!points.length) return { x: 0, y: 0, width: 1600, height: 1000 };
  const x = Math.min(...points.map((p) => p.x)),
    y = Math.min(...points.map((p) => p.y));
  return {
    x,
    y,
    width: Math.max(20, Math.max(...points.map((p) => p.x)) - x),
    height: Math.max(20, Math.max(...points.map((p) => p.y)) - y),
  };
}

export function moveSelection(
  scene: Scene,
  ids: string[],
  dx: number,
  dy: number,
): Scene {
  const selected = scene.elements.filter(
    (e) => ids.includes(e.id) && !e.locked && e.kind !== "arrow",
  );
  if (!selected.length) return scene;
  const box = bounds(selected);
  dx = Math.max(-box.x, Math.min(dx, 4000 - box.x - box.width));
  dy = Math.max(-box.y, Math.min(dy, 4000 - box.y - box.height));
  return {
    ...scene,
    elements: scene.elements.map((e) =>
      selected.includes(e) ? moveObject(e, dx, dy) : e,
    ),
  };
}

export function duplicateSelection(
  scene: Scene,
  ids: string[],
  source = scene.elements,
): { scene: Scene; ids: string[] } {
  const chosen = source.filter(
    (e) =>
      ids.includes(e.id) ||
      (e.kind === "arrow" &&
        ids.includes(e.from || "") &&
        ids.includes(e.to || "")),
  );
  const mapping = new Map(chosen.map((e) => [e.id, crypto.randomUUID()]));
  const groups = new Map(
    chosen.filter((e) => e.group).map((e) => [e.group!, crypto.randomUUID()]),
  );
  const copies = chosen
    .filter(
      (e) => e.kind !== "arrow" || (mapping.has(e.from!) && mapping.has(e.to!)),
    )
    .map((e) => ({
      ...moveObject(e, 30, 30),
      id: mapping.get(e.id)!,
      from: e.from ? mapping.get(e.from) : undefined,
      to: e.to ? mapping.get(e.to) : undefined,
      group: e.group ? groups.get(e.group) : undefined,
      locked: false,
    }));
  return {
    scene: { ...scene, elements: [...scene.elements, ...copies] },
    ids: copies.map((e) => e.id),
  };
}

export function wrapLabel(
  text: string,
  width: number,
  fontSize = 18,
): string[] {
  const limit = Math.max(1, Math.floor((width - 24) / (fontSize * 0.58)));
  const lines: string[] = [];
  for (const paragraph of text.split("\n")) {
    let line = "";
    for (const word of paragraph.split(/\s+/).filter(Boolean)) {
      if (!line) {
        if ([...word].length <= limit) line = word;
        else {
          for (const char of [...word]) {
            line += char;
            if ([...line].length >= limit) {
              lines.push(line);
              line = "";
            }
          }
        }
      } else if ([...line].length + 1 + [...word].length <= limit) {
        line += ` ${word}`;
      } else {
        lines.push(line);
        line = "";
        if ([...word].length <= limit) line = word;
        else
          for (const char of [...word]) {
            line += char;
            if ([...line].length >= limit) {
              lines.push(line);
              line = "";
            }
          }
      }
    }
    if (line || !paragraph) lines.push(line);
  }
  return lines;
}

export function download(content: string, type: string, filename: string) {
  const url = URL.createObjectURL(new Blob([content], { type }));
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function svgExport(svg: SVGSVGElement): string {
  const clone = svg.cloneNode(true) as SVGSVGElement;
  const originals = [svg, ...svg.querySelectorAll("*")];
  const copies = [clone, ...clone.querySelectorAll("*")];
  originals.forEach((node, i) => {
    const style = getComputedStyle(node);
    for (const name of [
      "fill",
      "stroke",
      "stroke-width",
      "font-family",
      "font-size",
      "text-anchor",
      "dominant-baseline",
      "stroke-linecap",
      "stroke-linejoin",
      "stroke-dasharray",
      "font-weight",
      "opacity",
    ])
      copies[i].setAttribute(
        name,
        style
          .getPropertyValue(name)
          .replace(/url\(["']?[^)"']*#([^)"']+)["']?\)/g, "url(#$1)"),
      );
    copies[i].removeAttribute("class");
    copies[i].removeAttribute("tabindex");
    copies[i].removeAttribute("style");
  });
  clone.querySelectorAll("[data-editor-only]").forEach((node) => node.remove());
  const paper = svg.querySelector<SVGRectElement>("[data-board-paper]");
  if (paper)
    clone.setAttribute(
      "viewBox",
      `0 0 ${paper.getAttribute("width")} ${paper.getAttribute("height")}`,
    );
  clone.setAttribute("xmlns", "http://www.w3.org/2000/svg");
  clone.setAttribute(
    "width",
    paper?.getAttribute("width") || String(svg.viewBox.baseVal.width),
  );
  clone.setAttribute(
    "height",
    paper?.getAttribute("height") || String(svg.viewBox.baseVal.height),
  );
  clone.removeAttribute("aria-describedby");
  return new XMLSerializer().serializeToString(clone);
}

export async function pngExport(svg: SVGSVGElement) {
  const source = new Blob([svgExport(svg)], { type: "image/svg+xml" });
  const url = URL.createObjectURL(source);
  try {
    const image = new Image();
    image.src = url;
    await image.decode();
    const canvas = document.createElement("canvas");
    const scale = Math.min(
      1,
      4096 / Math.max(image.naturalWidth, image.naturalHeight),
    );
    canvas.width = Math.max(1, Math.round(image.naturalWidth * scale));
    canvas.height = Math.max(1, Math.round(image.naturalHeight * scale));
    canvas
      .getContext("2d")!
      .drawImage(image, 0, 0, canvas.width, canvas.height);
    const blob = await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (b) => (b ? resolve(b) : reject(new Error("Could not export PNG."))),
        "image/png",
      ),
    );
    const downloadURL = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = downloadURL;
    a.download = "trestle-board.png";
    a.click();
    setTimeout(() => URL.revokeObjectURL(downloadURL), 1000);
  } finally {
    URL.revokeObjectURL(url);
  }
}

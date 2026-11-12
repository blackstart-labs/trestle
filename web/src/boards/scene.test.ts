import { expect, test } from "vitest";
import {
  arrowEnds,
  bounds,
  connectorPath,
  duplicateSelection,
  moveObject,
  moveSelection,
  removeObject,
  validateScene,
  wrapLabel,
  type BoardObject,
  type Scene,
} from "./scene";

const a: BoardObject = {
  id: "a",
  kind: "rectangle",
  x: 20,
  y: 20,
  width: 100,
  height: 60,
  text: "Start",
  color: "ink",
};
const b = { ...a, id: "b", x: 400 };
const arrow: BoardObject = {
  ...a,
  id: "edge",
  kind: "arrow",
  x: 0,
  y: 0,
  width: 0,
  height: 0,
  from: "a",
  to: "b",
};
const scene: Scene = { version: 1, elements: [a, b, arrow] };

test("validated portable scenes reject unsupported content and broken references", () => {
  expect(validateScene(scene)).toBe(true);
  for (const value of [
    null,
    {},
    { ...scene, version: 2 },
    { ...scene, elements: [a, a] },
    { ...scene, elements: [arrow] },
    { ...scene, elements: [{ ...a, color: "url(https://example.com)" }] },
    { ...scene, elements: [{ ...a, x: Infinity }] },
    { ...scene, elements: [{ ...a, text: "x".repeat(201) }] },
    {
      ...scene,
      elements: [{ ...a, kind: "stroke", points: [{ x: 0, y: 0 }] }],
    },
  ])
    expect(validateScene(value)).toBe(false);
});

test("removing a shape removes its attached arrows, not other shapes", () => {
  expect(removeObject(scene, "a").elements).toEqual([b]);
  expect(scene.elements).toHaveLength(3);
});

test("moving shapes and strokes clamps every point inside the board", () => {
  expect(moveObject(a, -100, -100)).toMatchObject({ x: 0, y: 0 });
  expect(moveObject(a, 9999, 9999)).toMatchObject({ x: 3900, y: 3940 });
  const stroke: BoardObject = {
    ...a,
    kind: "stroke",
    points: [
      { x: 10, y: 20 },
      { x: 30, y: 50 },
    ],
  };
  expect(moveObject(stroke, -100, 100).points).toEqual([
    { x: 0, y: 120 },
    { x: 20, y: 150 },
  ]);
});

test("connected arrows follow the boundary when their shapes move", () => {
  expect(arrowEnds(arrow, scene)).toEqual({
    from: { x: 120, y: 50 },
    to: { x: 400, y: 50 },
  });
  expect(
    arrowEnds(arrow, { ...scene, elements: [a, { ...b, x: 500 }, arrow] }).to.x,
  ).toBe(500);
});

test("extended formatting validates and new fields round-trip", () => {
  const fancy: BoardObject = {
    ...a,
    kind: "parallelogram",
    fill: "info",
    color: "info",
    strokeWidth: 4,
    dashed: true,
    fontSize: 24,
    bold: true,
    align: "left",
    group: "lane-1",
    locked: true,
  };
  expect(validateScene({ version: 1, elements: [fancy] })).toBe(true);
  for (const patch of [
    { fill: "url(#x)" },
    { strokeWidth: 0 },
    { strokeWidth: 9 },
    { fontSize: 8 },
    { fontSize: 80 },
    { align: "justify" },
    { route: "bezier" },
    { fromPort: "center" },
    { heads: "start" },
    { group: "g".repeat(65) },
    { bold: "yes" },
  ])
    expect(
      validateScene({ version: 1, elements: [{ ...fancy, ...patch }] }),
    ).toBe(false);
});

test("connector routing honours ports and styles", () => {
  const edge = { ...arrow, route: "elbow" as const };
  expect(connectorPath(edge, scene)).toBe("M120,50 H260 V50 H400");
  expect(connectorPath({ ...edge, route: "straight" }, scene)).toBe(
    "M120,50 L400,50",
  );
  expect(connectorPath({ ...edge, route: "curve" }, scene)).toBe(
    "M120,50 C260,50 260,50 400,50",
  );
  expect(
    arrowEnds({ ...edge, fromPort: "top", toPort: "left" }, scene),
  ).toEqual({ from: { x: 70, y: 20 }, to: { x: 400, y: 50 } });
});

test("bounds ignore arrows, follow strokes, and wrap long labels", () => {
  expect(bounds([a])).toEqual({ x: 20, y: 20, width: 100, height: 60 });
  expect(
    bounds([
      {
        ...a,
        kind: "stroke",
        points: [
          { x: 80, y: 90 },
          { x: 10, y: 30 },
        ],
      },
    ]),
  ).toEqual({ x: 10, y: 30, width: 70, height: 60 });
  expect(bounds([])).toEqual({ x: 0, y: 0, width: 1600, height: 1000 });
  expect(wrapLabel("abcdefghij", 100, 18)).toEqual(["abcdefg", "hij"]);
  expect(wrapLabel("the quick brown fox", 124, 18)).toEqual([
    "the quick",
    "brown fox",
  ]);
});

test("selection movement clamps and skips locked objects", () => {
  expect(moveSelection(scene, ["a"], -100, 100).elements[0]).toMatchObject({
    x: 0,
    y: 120,
  });
  expect(moveSelection(scene, ["a"], 9999, 0).elements[0]).toMatchObject({
    x: 3900,
  });
  expect(
    moveSelection(
      { ...scene, elements: [{ ...a, locked: true }, b, arrow] },
      ["a", "b"],
      10,
      0,
    ).elements,
  ).toEqual([{ ...a, locked: true }, { ...b, x: 410 }, arrow]);
});

test("duplicating a selection remaps attached arrows and groups", () => {
  const grouped: Scene = {
    ...scene,
    elements: [{ ...a, group: "g" }, { ...b, group: "g" }, arrow],
  };
  const result = duplicateSelection(grouped, ["a", "b"]);
  expect(result.scene.elements).toHaveLength(6);
  const copies = result.scene.elements.filter((e) => result.ids.includes(e.id));
  expect(copies).toHaveLength(3);
  const edge = copies.find((e) => e.kind === "arrow")!;
  expect(result.ids).toContain(edge.from!);
  expect(result.ids).toContain(edge.to!);
  const shapes = copies.filter((e) => e.kind !== "arrow");
  expect(shapes).toHaveLength(2);
  expect(shapes.every((e) => e.group && e.group !== "g")).toBe(true);
  expect(new Set(shapes.map((e) => e.group)).size).toBe(1);
});

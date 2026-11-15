import { describe, expect, it } from "vitest";
import { completionPercent } from "./identity";

describe("completionPercent", () => {
  it("counts only completed work against non-cancelled tasks", () => {
    expect(completionPercent(1, 2, 1)).toBe(100);
    expect(completionPercent(2, 5, 1)).toBe(50);
  });

  it("never divides by zero", () => {
    expect(completionPercent(0, 0, 0)).toBe(0);
    expect(completionPercent(0, 1, 1)).toBe(0);
  });

  it("ignores a cancelled count larger than the total", () => {
    expect(completionPercent(1, 1, 2)).toBe(0);
  });
});

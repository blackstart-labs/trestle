import { describe, expect, it } from "vitest";
import { documentLinks } from "./model";
describe("portable document links", () => {
  it("keeps UUID identities and deduplicates links", () => {
    const id = "123e4567-e89b-12d3-a456-426614174000";
    expect(documentLinks(`[[${id}|Old]] [[${id}|New]] [[bad]]`)).toEqual([
      { id, title: "New" },
    ]);
  });
  it("does not treat URLs or scripts as document identities", () => {
    expect(
      documentLinks("[[javascript:alert(1)]] [[https://example.com]]"),
    ).toEqual([]);
  });
});

import { expect, test } from "vitest";
import { renderMarkdown } from "./markdown";

test("renders lists, emphasis, links, code, tables and checklists", () => {
  const container = document.createElement("div");
  container.innerHTML = renderMarkdown(
    '## Plan\n\n- **Structure**\n- *Next step*\n\n[Reference](https://example.com)\n\n```go\nfmt.Println("hello")\n```\n\n| Task | Owner |\n| --- | --- |\n| Build | Ada |\n\n- [x] Done',
  );
  expect(container.querySelector("h2")?.textContent).toBe("Plan");
  expect(container.querySelector("strong")?.textContent).toBe("Structure");
  expect(container.querySelector("em")?.textContent).toBe("Next step");
  expect(container.querySelector("a")?.href).toBe("https://example.com/");
  expect(container.querySelector("a")?.target).toBe("_blank");
  expect(container.querySelector("a")?.rel).toBe("noopener noreferrer");
  expect(container.querySelector("pre code")?.textContent).toContain(
    'fmt.Println("hello")',
  );
  expect(container.querySelector("td")?.textContent).toBe("Build");
  expect(container.querySelector("input")?.disabled).toBe(true);
});

test("untrusted descriptions cannot execute HTML, load media, or use unsafe links", () => {
  const container = document.createElement("div");
  container.innerHTML = renderMarkdown(
    `<script>alert(1)</script>\n\n<img src=x onerror=alert(1)>\n\n<svg onload=alert(1)>\n\n![Private](https://external.example/image.png)\n\n[click](javascript:alert%281%29)\n\n[encoded](jav&#x61;script:alert%281%29)\n\n[data](data:text/html,bad)\n\n[quote](https://example.com/\"onclick=bad)`,
  );
  expect(container.querySelector("script, img, svg, iframe, style")).toBeNull();
  expect(container.textContent).toContain("<img src=x onerror=alert(1)>");
  expect(container.textContent).toContain("Private");
  for (const link of container.querySelectorAll("a")) {
    expect(link.getAttribute("href") || "").not.toMatch(/^(javascript|data):/i);
    expect(link.hasAttribute("onclick")).toBe(false);
  }
});

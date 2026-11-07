import { Marked } from "marked";
import DOMPurify from "dompurify";

const escape = (text: string) =>
  text.replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ]!,
  );

const markdown = new Marked({
  renderer: {
    // Descriptions are text, never an HTML or image upload surface.
    html: ({ text }) => escape(text),
    image: ({ text }) => escape(text),
    link({ href, title, tokens }) {
      // Following a reference must not navigate away from an unsaved draft.
      return `<a href="${escape(href)}" target="_blank" rel="noopener noreferrer"${title ? ` title="${escape(title)}"` : ""}>${this.parser.parseInline(tokens)}</a>`;
    },
    checkbox: ({ checked }) =>
      `<input type="checkbox" disabled aria-label="${checked ? "Completed item" : "Incomplete item"}"${checked ? " checked" : ""}>`,
  },
});

export function renderMarkdown(source: string): string {
  return DOMPurify.sanitize(markdown.parse(source, { async: false }), {
    ALLOWED_TAGS: [
      "p",
      "br",
      "strong",
      "em",
      "del",
      "h1",
      "h2",
      "h3",
      "h4",
      "h5",
      "h6",
      "ul",
      "ol",
      "li",
      "blockquote",
      "pre",
      "code",
      "a",
      "hr",
      "table",
      "thead",
      "tbody",
      "tr",
      "th",
      "td",
      "input",
    ],
    ALLOWED_ATTR: [
      "href",
      "title",
      "start",
      "type",
      "checked",
      "disabled",
      "aria-label",
      "target",
      "rel",
    ],
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
  });
}

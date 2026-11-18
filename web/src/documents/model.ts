export type DocumentPage = {
  id: string;
  workspace_id: string;
  title: string;
  content: string;
  tags: string[];
  parent_id: string | null;
  version: number;
  archived: boolean;
  updated_at: string;
};
export function documentLinks(content: string) {
  const links = new Map<string, string>();
  for (const m of content.matchAll(
    /\[\[([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})(?:\|([^\]\r\n]{0,160}))?\]\]/g,
  ))
    links.set(m[1].toLowerCase(), m[2] || m[1]);
  return [...links].map(([id, title]) => ({ id, title }));
}
export const templates: Record<string, string> = {
  blank: "",
  meeting:
    "# Meeting notes\n\n## Agenda\n\n- \n\n## Decisions\n\n## Action items\n\n- [ ] \n",
  project:
    "# Project brief\n\n## Outcome\n\n## Scope\n\n## Success criteria\n\n- [ ] \n\n## Risks\n\n## References\n",
};

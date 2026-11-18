<script lang="ts">
  import { onMount } from "svelte";
  import KnowledgeGraph from "./KnowledgeGraph.svelte";
  import Button from "../ui/Button.svelte";
  import Select from "../ui/Select.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Toast from "../ui/Toast.svelte";
  import Icon from "../ui/Icon.svelte";
  import Modal from "../ui/Modal.svelte";
  import MarkdownEditor from "../ui/MarkdownEditor.svelte";
  import { relativeTime } from "../lib/identity";
  import { documentLinks, templates, type DocumentPage } from "./model";

  let { workspace }: { workspace: { id: string; name: string; role: string } } =
    $props();

  const base = $derived(`/api/v1/workspaces/${workspace.id}/documents`);
  const canEdit = $derived(workspace.role !== "guest");
  const manager = $derived(["owner", "admin"].includes(workspace.role));

  let graphOpen = $state(false);
  let pages = $state<DocumentPage[]>([]);
  let current = $state<DocumentPage | null>(null);

  let title = $state("");
  let content = $state("");
  let tags = $state("");
  let parent = $state("");
  let baseline = $state("");
  let key = "";

  let treeQuery = $state("");
  let tagQuery = $state("");
  let archived = $state(false);
  let offset = $state(0);
  let more = $state(false);

  let loading = $state(false);
  let busy = $state(false);
  let error = $state("");
  let success = $state<string | null>(null);

  let history = $state<DocumentPage[]>([]);
  let historyOffset = $state(0);
  let historyMore = $state(false);
  let historyOpen = $state(false);

  let backlinks = $state<DocumentPage[]>([]);
  let backlinksOffset = $state(0);
  let backlinksMore = $state(false);

  let template = $state("blank");
  let parentQuery = $state("");
  let parentOptions = $state<DocumentPage[]>([]);

  let expandedIds = $state<Set<string>>(new Set());
  let showDeleteConfirm = $state(false);
  let deleting = $state(false);

  const draft = $derived(JSON.stringify({ title, content, tags, parent }));
  const dirty = $derived(!!current && draft !== baseline);
  const editable = $derived(canEdit && !current?.archived);
  const links = $derived(documentLinks(content));

  // Word count and reading time statistics
  const wordCount = $derived(
    content.trim() ? (content.trim().match(/\S+/g) || []).length : 0,
  );
  const charCount = $derived(content.length);
  const readTime = $derived(Math.max(1, Math.ceil(wordCount / 200)));

  // Hierarchical tree node structure for Trilium Notes style
  interface DocNode {
    doc: DocumentPage;
    children: DocNode[];
  }

  const docTree = $derived.by(() => {
    const filter = treeQuery.trim().toLowerCase();
    const map = new Map<string, DocNode>();
    for (const p of pages) {
      map.set(p.id, { doc: p, children: [] });
    }
    const roots: DocNode[] = [];
    for (const p of pages) {
      const node = map.get(p.id)!;
      if (p.parent_id && map.has(p.parent_id)) {
        map.get(p.parent_id)!.children.push(node);
      } else {
        roots.push(node);
      }
    }

    if (!filter) return roots;

    // Filter tree preserving matches and their ancestors
    function matchesFilter(node: DocNode): boolean {
      const selfMatch =
        node.doc.title.toLowerCase().includes(filter) ||
        node.doc.tags.some((t) => t.toLowerCase().includes(filter));
      const childMatches = node.children.some(matchesFilter);
      return selfMatch || childMatches;
    }

    function filterNode(node: DocNode): DocNode | null {
      if (!matchesFilter(node)) return null;
      return {
        doc: node.doc,
        children: node.children
          .map(filterNode)
          .filter((n): n is DocNode => n !== null),
      };
    }

    return roots.map(filterNode).filter((n): n is DocNode => n !== null);
  });

  // Calculate breadcrumb trail from root to current document
  const breadcrumbs = $derived.by(() => {
    if (!current || !current.id) return [];
    const trail: DocumentPage[] = [];
    const map = new Map(pages.map((p) => [p.id, p]));
    let pid = current.parent_id;
    const visited = new Set<string>();
    while (pid && map.has(pid) && !visited.has(pid)) {
      visited.add(pid);
      const parentPage = map.get(pid)!;
      trail.unshift(parentPage);
      pid = parentPage.parent_id;
    }
    return trail;
  });

  // Child documents of the currently opened document
  const currentChildren = $derived.by(() => {
    if (!current?.id) return [];
    return pages.filter((p) => p.parent_id === current?.id);
  });

  async function request(path: string, method = "GET", body?: unknown) {
    const csrf =
      document.cookie
        .split(";")
        .map((p) => p.trim())
        .find((p) => p.startsWith("trestle-csrf="))
        ?.slice(13) || "";
    let res: Response;
    try {
      res = await fetch(path, {
        method,
        cache: "no-store",
        headers: {
          Accept: "application/json",
          ...(body || method === "DELETE"
            ? { "Content-Type": "application/json", "X-CSRF-Token": csrf }
            : {}),
        },
        ...(body ? { body: JSON.stringify(body) } : {}),
      });
    } catch {
      throw new Error(
        "Could not reach the server. Your draft is unchanged. Check your connection and retry.",
      );
    }
    if (method === "DELETE" && res.status === 204) {
      return null;
    }
    const data = await res.json().catch(() => null);
    if (!res.ok || !data)
      throw new Error(
        data?.error?.message ||
          "Could not load or save documents. Check your access and retry.",
      );
    return data;
  }

  function failure(e: unknown) {
    error = e instanceof Error ? e.message : "The request failed. Try again.";
  }

  async function load() {
    loading = true;
    error = "";
    try {
      const data = await request(
        `${base}?${new URLSearchParams({
          q: "",
          tag: tagQuery,
          archived: String(archived),
          offset: String(offset),
        })}`,
      );
      pages = data.documents;
      more = data.has_more;
    } catch (e) {
      failure(e);
    } finally {
      loading = false;
    }
  }

  function setPage(page: DocumentPage) {
    current = page;
    title = page.title;
    content = page.content;
    tags = page.tags.join(", ");
    parent = page.parent_id || "";
    baseline = page.id ? JSON.stringify({ title, content, tags, parent }) : "";
    key = crypto.randomUUID();
    history = [];
    historyOpen = false;
    historyOffset = 0;
    backlinks = [];
    backlinksOffset = 0;
    backlinksMore = false;
    parentOptions = [];
    parentQuery = "";

    // Automatically expand all ancestor nodes in tree
    if (page.parent_id) {
      const next = new Set(expandedIds);
      let pid: string | null = page.parent_id;
      const map = new Map(pages.map((p) => [p.id, p]));
      while (pid && map.has(pid)) {
        next.add(pid);
        pid = map.get(pid)!.parent_id;
      }
      expandedIds = next;
    }
  }

  function canLeave() {
    return (
      !busy &&
      (!dirty ||
        confirm(
          "Discard unsaved document changes? Export Markdown first if you want to keep your draft.",
        ))
    );
  }

  async function loadBacklinks() {
    if (!current?.id) return;
    const data = await request(
      `${base}/${current.id}/backlinks?offset=${backlinksOffset}`,
    );
    backlinks = data.documents;
    backlinksMore = data.has_more;
  }

  async function open(id: string) {
    if (!canLeave()) return;
    busy = true;
    error = "";
    try {
      setPage(await request(`${base}/${id}`));
      await loadBacklinks();
    } catch (e) {
      failure(e);
    } finally {
      busy = false;
    }
  }

  function create(parentId = "") {
    if (!canLeave()) return;
    setPage({
      id: "",
      workspace_id: workspace.id,
      title: "Untitled document",
      content: templates[template] || "",
      tags: [],
      parent_id: parentId || null,
      version: 0,
      archived: false,
      updated_at: "",
    });
    error = "";
    if (parentId) {
      const next = new Set(expandedIds);
      next.add(parentId);
      expandedIds = next;
    }
  }

  async function save(archive = current?.archived || false) {
    if (!current || busy) return;
    busy = true;
    error = "";
    try {
      const saved = await request(
        current.id ? `${base}/${current.id}` : base,
        current.id ? "PUT" : "POST",
        {
          title,
          content,
          tags: [
            ...new Set(
              tags
                .split(",")
                .map((t) => t.trim())
                .filter(Boolean),
            ),
          ],
          parent_id: parent || null,
          version: current.version,
          archived: archive,
          creation_key: key,
        },
      );
      setPage(saved);
      success = archive
        ? "Document archived. Child pages remain available."
        : "Document saved.";
      await load();
      await loadBacklinks();
    } catch (e) {
      failure(e);
    } finally {
      busy = false;
    }
  }

  async function deleteCurrentDocument() {
    if (!current?.id || deleting) return;
    deleting = true;
    error = "";
    try {
      await request(`${base}/${current.id}`, "DELETE");
      success = "Document deleted.";
      showDeleteConfirm = false;
      const deletedId = current.id;
      current = null;
      await load();
      if (pages.length > 0) {
        const next = pages.find((p) => p.id !== deletedId);
        if (next) void open(next.id);
      }
    } catch (e) {
      failure(e);
    } finally {
      deleting = false;
    }
  }

  async function loadHistory() {
    if (!current?.id || busy) return;
    busy = true;
    error = "";
    try {
      const data = await request(
        `${base}/${current.id}/versions?offset=${historyOffset}`,
      );
      history = data.documents;
      historyMore = data.has_more;
      historyOpen = true;
    } catch (e) {
      failure(e);
    } finally {
      busy = false;
    }
  }

  async function restore(version: number) {
    if (
      !current ||
      !canLeave() ||
      !confirm(
        `Load revision ${version} into the editor? Save it as a new revision to restore.`,
      )
    )
      return;
    busy = true;
    error = "";
    try {
      const old = await request(`${base}/${current.id}/versions/${version}`);
      title = old.title;
      content = old.content;
      tags = old.tags.join(", ");
      parent = old.parent_id || "";
      success = "Revision loaded as a draft. Review it and save to restore.";
    } catch (e) {
      failure(e);
    } finally {
      busy = false;
    }
  }

  function exportMarkdown() {
    const url = URL.createObjectURL(
      new Blob([content], { type: "text/markdown;charset=utf-8" }),
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = `${title.replace(/[^\p{L}\p{N}_-]/gu, "_").slice(0, 100) || "document"}.md`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  async function importMarkdown(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    if (!canLeave()) {
      input.value = "";
      return;
    }
    try {
      if (file.size > 400000 || !/\.(md|markdown|txt)$/i.test(file.name))
        throw new Error("Choose a Markdown or text file up to 400 KB.");
      const text = await file.text();
      if ([...text].length > 100000 || text.includes("\0"))
        throw new Error("Use at most 100000 characters of valid text.");
      setPage({
        id: "",
        workspace_id: workspace.id,
        title: file.name.replace(/\.(md|markdown|txt)$/i, "").slice(0, 160),
        content: text,
        tags: [],
        parent_id: current?.id || null,
        version: 0,
        archived: false,
        updated_at: "",
      });
    } catch (e) {
      failure(e);
    } finally {
      input.value = "";
    }
  }

  async function searchParents() {
    if (busy) return;
    busy = true;
    error = "";
    try {
      const data = await request(
        `${base}?q=${encodeURIComponent(parentQuery)}`,
      );
      parentOptions = data.documents.filter(
        (d: DocumentPage) => d.id !== current?.id,
      );
    } catch (e) {
      failure(e);
    } finally {
      busy = false;
    }
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(
        `[[${current?.id}|${title.replace(/[\]\r\n]/g, " ")}]]`,
      );
      success = "Document link copied.";
    } catch {
      error = "Clipboard unavailable. Copy the document ID below.";
    }
  }

  function toggleExpand(id: string, e?: MouseEvent) {
    e?.stopPropagation();
    const next = new Set(expandedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    expandedIds = next;
  }

  onMount(() => {
    void load();
    const unload = (e: BeforeUnloadEvent) => {
      if (dirty || busy) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    const navigate = (e: Event) => {
      if (!canLeave()) e.preventDefault();
    };
    window.addEventListener("beforeunload", unload);
    window.addEventListener("trestle:before-navigate", navigate);
    return () => {
      window.removeEventListener("beforeunload", unload);
      window.removeEventListener("trestle:before-navigate", navigate);
    };
  });
</script>

<div class="page-head">
  <div>
    <h2>Documents</h2>
    <p>
      Connected knowledge tree and notes in {workspace.name}. Portable Markdown
      with bi-directional links.
    </p>
  </div>
  <div class="page-actions">
    <Button
      disabled={busy}
      onclick={() => {
        if (canLeave()) {
          graphOpen = !graphOpen;
          if (!graphOpen) void load();
        }
      }}
    >
      <Icon name={graphOpen ? "layers" : "workflow"} size={16} />
      {graphOpen ? "Document tree" : "Knowledge graph"}
    </Button>
    {#if current}
      <Button onclick={exportMarkdown}>
        <Icon name="download" size={16} />
        Export Markdown
      </Button>
      {#if editable}
        <Button
          variant="primary"
          disabled={busy || !dirty}
          onclick={() => save()}
        >
          <Icon name="check" size={16} />
          {busy ? "Saving…" : "Save document"}
        </Button>
      {/if}
    {:else if canEdit}
      <Select
        label="Document template"
        value={template}
        options={[
          { value: "blank", label: "Blank page" },
          { value: "meeting", label: "Meeting notes" },
          { value: "project", label: "Project brief" },
        ]}
        onchange={(v) => (template = v)}
      />
      <Button variant="primary" disabled={loading} onclick={() => create()}>
        <Icon name="plus" size={16} />
        New document
      </Button>
      <label class="btn import-button">
        <Icon name="upload" size={16} />
        Import Markdown
        <input
          aria-label="Import Markdown"
          type="file"
          accept=".md,.markdown,.txt"
          disabled={loading}
          onchange={importMarkdown}
        />
      </label>
    {/if}
  </div>
</div>

{#if error}<Feedback tone="danger" announcement="alert">{error}</Feedback>{/if}
{#if !canEdit}
  <Feedback>
    Guests can read and export documents. Ask a Workspace owner for editing
    access.
  </Feedback>
{/if}

{#if graphOpen}
  <KnowledgeGraph
    {workspace}
    onopen={(id) => {
      graphOpen = false;
      void open(id);
    }}
  />
{:else}
  <!-- Trilium Notes-Inspired 3-Pane Architecture -->
  <div class="trilium-workspace">
    <!-- Left Pane: Hierarchical Tree Navigation -->
    <aside class="trilium-tree card" aria-label="Document hierarchy">
      <div class="tree-header">
        <div class="tree-search-bar">
          <Icon name="search" size={14} class="tree-search-icon" />
          <input
            type="search"
            class="tree-search-input"
            placeholder="Filter notes…"
            bind:value={treeQuery}
            aria-label="Filter documents by title or tag"
          />
        </div>
        {#if canEdit}
          <button
            type="button"
            class="btn-quiet tree-btn-icon"
            title="Create root document"
            aria-label="Create root document"
            onclick={() => create()}
          >
            <Icon name="plus" size={16} />
          </button>
        {/if}
      </div>

      <div class="tree-body">
        {#if loading && !pages.length}
          <div class="tree-loading" aria-hidden="true">
            <span class="skeleton tree-skeleton-item"></span>
            <span class="skeleton tree-skeleton-item"></span>
            <span class="skeleton tree-skeleton-item"></span>
          </div>
        {:else if !pages.length}
          <div class="empty tree-empty">
            <span class="tile empty-mark"><Icon name="layers" /></span>
            <h3>Build your knowledge base</h3>
            <p>Create connected documents in portable Markdown.</p>
          </div>
        {:else}
          <nav class="tree-nav" aria-label="Note tree">
            {#snippet renderTreeNodes(nodes: DocNode[], depth: number)}
              <ul class="tree-list" role="tree">
                {#each nodes as node (node.doc.id)}
                  {@const hasChildren = node.children.length > 0}
                  {@const isExpanded = expandedIds.has(node.doc.id)}
                  {@const isSelected = current?.id === node.doc.id}
                  <li
                    role="treeitem"
                    aria-selected={isSelected}
                    aria-expanded={hasChildren ? isExpanded : undefined}
                    class="tree-item"
                  >
                    <div
                      class="tree-row"
                      class:selected={isSelected}
                      style:padding-left={`${depth * 14 + 6}px`}
                    >
                      {#if hasChildren}
                        <button
                          type="button"
                          class="tree-expander"
                          aria-label={isExpanded
                            ? "Collapse note"
                            : "Expand note"}
                          onclick={(e) => toggleExpand(node.doc.id, e)}
                        >
                          <Icon
                            name={isExpanded ? "chevron-down" : "chevron-right"}
                            size={12}
                          />
                        </button>
                      {:else}
                        <span class="tree-expander-spacer"></span>
                      {/if}

                      <button
                        type="button"
                        class="tree-node-link"
                        onclick={() => open(node.doc.id)}
                      >
                        <Icon
                          name={hasChildren
                            ? isExpanded
                              ? "folder"
                              : "folder"
                            : "document"}
                          size={15}
                          class="tree-node-icon"
                        />
                        <span class="tree-node-title">
                          {node.doc.title || "Untitled document"}
                        </span>
                        {#if hasChildren}
                          <span class="tree-node-count"
                            >{node.children.length}</span
                          >
                        {/if}
                      </button>

                      {#if canEdit}
                        <button
                          type="button"
                          class="tree-row-action"
                          title="Add sub-note"
                          aria-label={`Add sub-note under ${node.doc.title}`}
                          onclick={(e) => {
                            e.stopPropagation();
                            create(node.doc.id);
                          }}
                        >
                          <Icon name="plus" size={12} />
                        </button>
                      {/if}
                    </div>

                    {#if hasChildren && isExpanded}
                      {@render renderTreeNodes(node.children, depth + 1)}
                    {/if}
                  </li>
                {/each}
              </ul>
            {/snippet}

            {@render renderTreeNodes(docTree, 0)}
          </nav>
        {/if}
      </div>
    </aside>

    <!-- Center Pane: Document Reader & Editor -->
    <main class="trilium-editor card" aria-label="Document editor">
      {#if current}
        <header class="editor-header">
          <!-- Trilium Breadcrumb Ancestry Trail -->
          <nav aria-label="Breadcrumb" class="breadcrumbs">
            <button
              type="button"
              class="breadcrumb-root"
              onclick={() => {
                if (canLeave()) current = null;
              }}
            >
              <Icon name="layers" size={13} />
              <span>Documents</span>
            </button>
            {#each breadcrumbs as ancestor}
              <span class="breadcrumb-sep">/</span>
              <button
                type="button"
                class="breadcrumb-item"
                onclick={() => open(ancestor.id)}
              >
                {ancestor.title}
              </button>
            {/each}
            <span class="breadcrumb-sep">/</span>
            <span class="breadcrumb-current">
              {title || "Untitled document"}
            </span>
          </nav>

          <!-- Document Title Field -->
          <div class="field title-field">
            <label for="document-title" class="visually-hidden">Title</label>
            <input
              id="document-title"
              class="title-input"
              bind:value={title}
              placeholder="Untitled document"
              maxlength="160"
              disabled={busy || !editable}
            />
          </div>

          <!-- Document Metadata & Status Ribbon -->
          <div class="editor-status-bar">
            <div class="status-tags">
              {#if current.id}
                <span class="meta-pill">v{current.version}</span>
              {:else}
                <span class="meta-pill new-pill">Draft</span>
              {/if}
              {#if dirty}
                <span class="status-chip chip-warning">Unsaved changes</span>
              {:else if current.id}
                <span class="status-chip chip-success">Saved</span>
              {/if}
              {#if current.archived}
                <span class="status-chip chip-neutral">Archived</span>
              {/if}
            </div>

            <div class="reading-stats">
              <span>{wordCount} words</span>
              <span>·</span>
              <span>~{readTime} min read</span>
            </div>
          </div>
        </header>

        <!-- Markdown Editor -->
        <div class="editor-canvas">
          <MarkdownEditor
            id="document-content"
            label="Document content"
            bind:value={content}
            disabled={busy}
            readonly={!editable}
          />
        </div>

        {#if links.length}
          <section aria-label="Linked documents" class="linked-section">
            <div class="linked-head">
              <Icon name="workflow" size={14} />
              <h4>Linked documents ({links.length})</h4>
            </div>
            <div class="linked-grid">
              {#each links as link}
                <button
                  type="button"
                  class="linked-card"
                  disabled={busy}
                  onclick={() => open(link.id)}
                >
                  <Icon name="document" size={14} />
                  <span>{link.title}</span>
                </button>
              {/each}
            </div>
          </section>
        {/if}
      {:else}
        <div class="editor-empty">
          <span class="tile empty-mark"><Icon name="document" size={28} /></span
          >
          <h3>Select a note to read or edit</h3>
          <p>
            Choose a document from the tree on the left, or create a new one to
            begin writing.
          </p>
        </div>
      {/if}
    </main>

    <!-- Right Pane: Attributes & Relations Inspector -->
    {#if current}
      <aside
        class="trilium-inspector card"
        aria-label="Document attributes and context"
      >
        <div class="inspector-section">
          <h3 class="inspector-heading">Attributes</h3>
          <div class="field">
            <label for="document-tags">Tags (comma separated)</label>
            <input
              id="document-tags"
              bind:value={tags}
              placeholder="e.g. engineering, architecture"
              disabled={busy || !editable}
            />
            <small class="field-hint">Up to 20 tags, 40 characters each.</small>
          </div>

          <div class="stat-rows">
            <div class="stat-row">
              <span class="stat-label">Characters</span>
              <span class="stat-value">{charCount}</span>
            </div>
            <div class="stat-row">
              <span class="stat-label">Words</span>
              <span class="stat-value">{wordCount}</span>
            </div>
            {#if current.updated_at}
              <div class="stat-row">
                <span class="stat-label">Updated</span>
                <span class="stat-value"
                  >{relativeTime(current.updated_at)}</span
                >
              </div>
            {/if}
          </div>
        </div>

        <!-- Hierarchy & Relations Section -->
        <div class="inspector-section">
          <h3 class="inspector-heading">Hierarchy</h3>
          {#if parent}
            <Button
              variant="secondary"
              disabled={busy}
              onclick={() => open(parent)}
            >
              <Icon name="arrow-left" size={14} />
              Open parent note
            </Button>
          {/if}

          {#if editable}
            <div class="parent-picker">
              <div class="field">
                <label for="parent-search">Change parent note</label>
                <div class="search-input-group">
                  <input
                    id="parent-search"
                    bind:value={parentQuery}
                    placeholder="Search note title…"
                    maxlength="200"
                  />
                  <Button disabled={busy} onclick={searchParents}>
                    Search
                  </Button>
                </div>
              </div>

              {#if parentOptions.length > 0 || parent}
                <Select
                  label="Selected parent"
                  value={parent}
                  disabled={busy}
                  options={[
                    { value: "", label: "Root (Top-level)" },
                    ...(parent && !parentOptions.some((p) => p.id === parent)
                      ? [{ value: parent, label: "Current parent" }]
                      : []),
                    ...parentOptions.map((p) => ({
                      value: p.id,
                      label: p.title,
                    })),
                  ]}
                  onchange={(v) => (parent = v)}
                />
              {/if}
            </div>

            <Button
              variant="secondary"
              disabled={busy}
              onclick={() => create(current?.id)}
            >
              <Icon name="plus" size={14} />
              Add child note
            </Button>
          {/if}

          {#if currentChildren.length > 0}
            <div class="subnotes-box">
              <span class="subnotes-title"
                >Direct Sub-notes ({currentChildren.length})</span
              >
              <div class="subnotes-list">
                {#each currentChildren as child}
                  <button
                    type="button"
                    class="subnote-link"
                    onclick={() => open(child.id)}
                  >
                    <Icon name="document" size={13} />
                    <span>{child.title}</span>
                  </button>
                {/each}
              </div>
            </div>
          {/if}
        </div>

        <!-- Backlinks / Incoming References -->
        <div class="inspector-section">
          <h3 class="inspector-heading">Backlinks ({backlinks.length})</h3>
          {#if backlinks.length > 0}
            <div class="backlinks-list">
              {#each backlinks as bl}
                <button
                  type="button"
                  class="backlink-card"
                  disabled={busy}
                  onclick={() => open(bl.id)}
                >
                  <Icon name="workflow" size={13} />
                  <span>{bl.title}</span>
                </button>
              {/each}
            </div>
          {:else}
            <p class="empty-note">No incoming links to this note.</p>
          {/if}
        </div>

        <!-- Document Actions -->
        <div class="inspector-section actions-section">
          <h3 class="inspector-heading">Actions</h3>
          {#if current.id}
            <Button disabled={busy} onclick={copyLink}>
              <Icon name="copy" size={14} />
              Copy note link
            </Button>
            <Button disabled={busy} onclick={loadHistory}>
              <Icon name="clock" size={14} />
              Revision history
            </Button>
            {#if manager}
              <Button
                variant={current.archived ? "secondary" : "danger-quiet"}
                disabled={busy || dirty}
                onclick={() => save(!current?.archived)}
              >
                <Icon name="archive" size={14} />
                {current.archived ? "Restore document" : "Archive document"}
              </Button>
              <Button
                variant="danger-quiet"
                disabled={busy}
                onclick={() => (showDeleteConfirm = true)}
              >
                <Icon name="trash" size={14} />
                Delete document
              </Button>
            {/if}
          {/if}
        </div>
      </aside>
    {/if}
  </div>
{/if}

<!-- Revision History Modal -->
{#if historyOpen}
  <Modal
    title={`Revisions of “${current?.title}”`}
    description="Review earlier versions and load any draft back into the editor."
    onclose={() => (historyOpen = false)}
  >
    <div class="history-list">
      {#each history as rev}
        <div class="history-row">
          <div class="history-info">
            <span class="history-ver">v{rev.version}</span>
            <span class="history-title">{rev.title}</span>
            <span class="history-date">{relativeTime(rev.updated_at)}</span>
          </div>
          {#if editable}
            <Button
              variant="secondary"
              disabled={busy}
              onclick={() => {
                historyOpen = false;
                void restore(rev.version);
              }}
            >
              Load revision {rev.version}
            </Button>
          {/if}
        </div>
      {:else}
        <p class="empty-note">No revision snapshots recorded yet.</p>
      {/each}
    </div>
  </Modal>
{/if}

<!-- Delete Confirmation Modal -->
{#if showDeleteConfirm && current}
  <Modal
    title="Delete document"
    description={`Are you sure you want to delete “${current.title}”? Sub-notes will be moved to its parent. This action cannot be undone.`}
    onclose={() => (showDeleteConfirm = false)}
  >
    <div class="dialog-actions">
      <Button
        variant="quiet"
        disabled={deleting}
        onclick={() => (showDeleteConfirm = false)}
      >
        Cancel
      </Button>
      <Button
        variant="danger"
        disabled={deleting}
        onclick={deleteCurrentDocument}
      >
        {deleting ? "Deleting…" : "Delete document"}
      </Button>
    </div>
  </Modal>
{/if}

{#if success}<Toast message={success} onclear={() => (success = null)} />{/if}

<style>
  .trilium-workspace {
    display: grid;
    grid-template-columns: 280px minmax(0, 1fr) 280px;
    gap: var(--space-4);
    min-height: calc(100vh - 180px);
    align-items: start;
    margin-top: var(--space-3);
  }

  @media (max-width: 1100px) {
    .trilium-workspace {
      grid-template-columns: 260px minmax(0, 1fr);
    }
    .trilium-inspector {
      grid-column: 1 / -1;
    }
  }

  @media (max-width: 768px) {
    .trilium-workspace {
      grid-template-columns: 1fr;
    }
  }

  /* ── Tree Sidebar ─────────────────────────────────────────────── */
  .trilium-tree {
    background: var(--surface-float);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    display: flex;
    flex-direction: column;
    min-height: 520px;
    max-height: calc(100vh - 180px);
    overflow: hidden;
    box-shadow: var(--shadow-sm);
  }

  .tree-header {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3);
    border-bottom: 1px solid var(--line);
    background: var(--surface-raised);
  }

  .tree-search-bar {
    position: relative;
    flex: 1;
    display: flex;
    align-items: center;
  }

  :global(.tree-search-icon) {
    position: absolute;
    left: 8px;
    color: var(--text-muted);
    pointer-events: none;
  }

  .tree-search-input {
    width: 100%;
    padding: 6px 10px 6px 28px;
    font-size: var(--text-xs);
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
  }

  .tree-btn-icon {
    width: 30px;
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-sm);
    flex-shrink: 0;
    color: var(--text-secondary);
  }

  .tree-body {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-2) 0;
  }

  .tree-empty {
    padding: var(--space-5) var(--space-3);
    text-align: center;
  }

  .tree-loading {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3);
  }

  .tree-skeleton-item {
    height: 28px;
    border-radius: var(--radius-sm);
  }

  .tree-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .tree-item {
    display: block;
  }

  .tree-row {
    display: flex;
    align-items: center;
    min-height: 34px;
    padding-right: var(--space-2);
    border-radius: var(--radius-sm);
    margin: 1px var(--space-1);
    color: var(--text-secondary);
    transition: background var(--duration-fast) var(--ease-out);
  }

  .tree-row:hover {
    background: var(--surface-raised);
    color: var(--text);
  }

  .tree-row.selected {
    background: var(--accent-soft);
    color: var(--accent);
    font-weight: 580;
  }

  .tree-expander {
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    color: var(--text-muted);
    border-radius: 3px;
    cursor: pointer;
    flex-shrink: 0;
  }

  .tree-expander-spacer {
    width: 20px;
    flex-shrink: 0;
  }

  .tree-node-link {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 6px;
    border: none;
    background: transparent;
    color: inherit;
    font-size: var(--text-sm);
    text-align: left;
    cursor: pointer;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    padding: 4px 0;
  }

  :global(.tree-node-icon) {
    flex-shrink: 0;
    color: var(--text-muted);
  }

  .tree-row.selected :global(.tree-node-icon) {
    color: var(--accent);
  }

  .tree-node-title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tree-node-count {
    font-size: 11px;
    color: var(--text-muted);
    background: var(--surface-raised);
    border-radius: var(--radius-pill);
    padding: 1px 6px;
    margin-left: auto;
    flex-shrink: 0;
  }

  .tree-row-action {
    width: 22px;
    height: 22px;
    display: none;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    color: var(--text-muted);
    border-radius: 3px;
    cursor: pointer;
    margin-left: 2px;
    flex-shrink: 0;
  }

  .tree-row:hover .tree-row-action {
    display: flex;
  }

  .tree-row-action:hover {
    background: var(--surface-raised);
    color: var(--text);
  }

  /* ── Center Editor Pane ────────────────────────────────────────── */
  .trilium-editor {
    background: var(--surface-float);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
    display: flex;
    flex-direction: column;
    min-height: 520px;
    padding: var(--space-4) var(--space-5);
    gap: var(--space-4);
  }

  .editor-header {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .breadcrumbs {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px;
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .breadcrumb-root,
  .breadcrumb-item {
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 2px 4px;
    border-radius: 3px;
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .breadcrumb-root:hover,
  .breadcrumb-item:hover {
    background: var(--surface-raised);
    color: var(--text);
  }

  .breadcrumb-sep {
    color: var(--line-strong);
  }

  .breadcrumb-current {
    font-weight: 600;
    color: var(--text);
  }

  .title-field {
    margin: var(--space-1) 0;
  }

  .title-input {
    font-size: var(--text-title);
    font-weight: 650;
    letter-spacing: -0.02em;
    color: var(--text);
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    padding: 4px 6px;
    width: 100%;
    background: transparent;
  }

  .title-input:focus {
    background: var(--surface);
    border-color: var(--border);
    outline: none;
  }

  .editor-status-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-bottom: var(--space-2);
    border-bottom: 1px solid var(--line);
    font-size: var(--text-xs);
  }

  .status-tags {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .meta-pill {
    padding: 2px 8px;
    border-radius: var(--radius-pill);
    background: var(--surface-raised);
    color: var(--text-secondary);
    font-weight: 500;
    font-size: 11px;
  }

  .meta-pill.new-pill {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .status-chip {
    padding: 2px 8px;
    border-radius: var(--radius-pill);
    font-size: 11px;
    font-weight: 600;
  }

  .chip-warning {
    background: var(--warning-soft);
    color: var(--warning);
  }

  .chip-success {
    background: var(--success-soft);
    color: var(--success);
  }

  .chip-neutral {
    background: var(--surface-raised);
    color: var(--text-muted);
  }

  .reading-stats {
    color: var(--text-muted);
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: var(--text-xs);
  }

  .editor-canvas {
    flex: 1;
    display: flex;
    flex-direction: column;
  }

  .editor-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    flex: 1;
    padding: var(--space-6);
    text-align: center;
    color: var(--text-secondary);
  }

  .editor-empty h3 {
    margin-top: var(--space-3);
    color: var(--text);
  }

  .editor-empty p {
    max-width: 24rem;
    margin: var(--space-2) 0 var(--space-4);
    font-size: var(--text-sm);
  }

  /* Linked documents section */
  .linked-section {
    margin-top: var(--space-4);
    padding-top: var(--space-3);
    border-top: 1px solid var(--line);
  }

  .linked-head {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: var(--space-2);
    color: var(--text-muted);
  }

  .linked-head h4 {
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin: 0;
  }

  .linked-grid {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }

  .linked-card {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 10px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--line);
    background: var(--surface-raised);
    color: var(--text);
    font-size: var(--text-xs);
    cursor: pointer;
  }

  .linked-card:hover {
    border-color: var(--accent);
    color: var(--accent);
  }

  /* ── Right Inspector Pane ───────────────────────────────────────── */
  .trilium-inspector {
    background: var(--surface-float);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .inspector-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding-bottom: var(--space-3);
    border-bottom: 1px solid var(--line);
  }

  .inspector-section:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .inspector-heading {
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
    margin: 0 0 4px;
  }

  .stat-rows {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-top: var(--space-2);
  }

  .stat-row {
    display: flex;
    justify-content: space-between;
    font-size: var(--text-xs);
  }

  .stat-label {
    color: var(--text-muted);
  }

  .stat-value {
    color: var(--text);
    font-weight: 500;
  }

  .field-hint {
    font-size: 11px;
    color: var(--text-muted);
  }

  .search-input-group {
    display: flex;
    gap: 4px;
  }

  .subnotes-box {
    margin-top: var(--space-2);
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .subnotes-title {
    font-size: 11px;
    color: var(--text-muted);
    font-weight: 600;
  }

  .subnotes-list,
  .backlinks-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .subnote-link,
  .backlink-card {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 8px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--line);
    background: var(--surface-raised);
    color: var(--text-secondary);
    font-size: var(--text-xs);
    cursor: pointer;
    text-align: left;
  }

  .subnote-link:hover,
  .backlink-card:hover {
    color: var(--accent);
    border-color: var(--accent);
  }

  .empty-note {
    font-size: var(--text-xs);
    color: var(--text-muted);
    margin: 0;
  }

  .actions-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  /* History Modal Styles */
  .history-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    max-height: 400px;
    overflow-y: auto;
  }

  .history-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    background: var(--surface-raised);
  }

  .history-info {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .history-ver {
    font-weight: 600;
    color: var(--accent);
  }

  .history-date {
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .import-button {
    position: relative;
    overflow: hidden;
    cursor: pointer;
  }

  .import-button input {
    position: absolute;
    inset: 0;
    width: 100%;
    opacity: 0;
    cursor: pointer;
  }
</style>

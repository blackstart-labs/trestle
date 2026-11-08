<script lang="ts">
  import { tick } from "svelte";
  import type { Workspace } from "./auth/WorkspaceChooser.svelte";
  import type { Project } from "./projects/ProjectsView.svelte";
  import Button from "./ui/Button.svelte";
  import Feedback from "./ui/Feedback.svelte";
  import Icon from "./ui/Icon.svelte";
  import Modal from "./ui/Modal.svelte";
  import Select from "./ui/Select.svelte";
  import Toast from "./ui/Toast.svelte";
  import { relativeTime } from "./lib/identity";

  type Connection = {
    provider: string;
    chat_id: string;
    topic_id?: string;
    last_attempt_at: string | null;
    last_error: string;
  };
  let {
    workspace,
    signedIn = false,
    initialProjectId = "",
    onselectworkspace,
  }: {
    workspace: Workspace | null;
    signedIn?: boolean;
    initialProjectId?: string;
    onselectworkspace?: (ws: Workspace) => void;
  } = $props();

  let internalWorkspace = $state<Workspace | null>(null);
  let availableWorkspaces = $state<Workspace[]>([]);
  let loadingWorkspace = $state(false);

  const activeWorkspace = $derived.by(() => {
    if (workspace) return workspace;
    return internalWorkspace;
  });

  $effect(() => {
    if (workspace) {
      internalWorkspace = null;
      return;
    }
    if (!signedIn) return;
    const controller = new AbortController();
    loadingWorkspace = true;
    fetch("/api/v1/workspaces", {
      headers: { Accept: "application/json" },
      signal: controller.signal,
    })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (!data || controller.signal.aborted) return;
        availableWorkspaces = data.workspaces || [];
        if (availableWorkspaces.length > 0) {
          const lastId = (() => {
            try {
              return localStorage.getItem("trestle.workspace");
            } catch {
              return null;
            }
          })();
          const target =
            availableWorkspaces.find((w: Workspace) => w.id === lastId) ||
            availableWorkspaces[0];
          internalWorkspace = target;
          onselectworkspace?.(target);
        }
      })
      .catch(() => {})
      .finally(() => {
        if (!controller.signal.aborted) loadingWorkspace = false;
      });
    return () => controller.abort();
  });
  const providers = ["telegram", "slack", "discord"] as const;
  const names: Record<string, string> = {
    telegram: "Telegram",
    slack: "Slack",
    discord: "Discord",
  };
  const descriptions: Record<string, string> = {
    telegram: "Send updates to a chat or channel with a bot.",
    slack: "Post updates to a channel through an incoming webhook.",
    discord: "Keep a server channel informed through a webhook.",
  };
  let projects = $state<Project[]>([]);
  let projectId = $state("");
  let connections = $state<Connection[]>([]);
  let loadingProjects = $state(false);
  let loading = $state(false);
  let loadError = $state("");
  let actionError = $state("");
  let message = $state<string | null>(null);
  let retry = $state(0);
  let editing = $state("");
  let secret = $state("");
  let chat = $state("");
  let topic = $state("");
  let originalChat = $state("");
  let originalTopic = $state("");
  let busy = $state(false);
  let disconnecting = $state("");
  const canManage = $derived(
    activeWorkspace?.role === "owner" || activeWorkspace?.role === "admin",
  );
  const project = $derived(projects.find((p) => p.id === projectId));
  const base = $derived(
    `/api/v1/workspaces/${activeWorkspace?.id}/projects/${projectId}/integrations`,
  );
  const dirty = $derived(
    secret !== "" || chat !== originalChat || topic !== originalTopic,
  );

  async function request(
    url: string,
    method = "GET",
    body?: object,
    signal?: AbortSignal,
  ) {
    const csrf = document.cookie
      .split(";")
      .map((v) => v.trim())
      .find((v) => v.startsWith("trestle-csrf="))
      ?.slice(13);
    const response = await fetch(url, {
      method,
      signal,
      cache: "no-store",
      headers: {
        Accept: "application/json",
        ...(body ? { "Content-Type": "application/json" } : {}),
        ...(csrf ? { "X-CSRF-Token": csrf } : {}),
      },
      ...(body ? { body: JSON.stringify(body) } : {}),
    });
    if (response.status === 204) return null;
    const data = await response.json();
    if (!response.ok)
      throw new Error(data.error?.message || "Request failed. Try again.");
    return data;
  }

  $effect(() => {
    const ws = activeWorkspace;
    const preferredProjectId = initialProjectId;
    void retry;
    projects = [];
    if (!ws || !canManage) return;
    const controller = new AbortController();
    loadingProjects = true;
    loadError = "";
    request(
      `/api/v1/workspaces/${ws.id}/projects`,
      "GET",
      undefined,
      controller.signal,
    )
      .then((data) => {
        if (controller.signal.aborted) return;
        projects = data.projects.filter((p: Project) => !p.archived);
        if (projects.some((p) => p.id === preferredProjectId))
          projectId = preferredProjectId;
        else if (!projects.some((p) => p.id === projectId))
          projectId = projects[0]?.id || "";
      })
      .catch(() => {
        if (!controller.signal.aborted)
          loadError = "Could not load projects. Try again.";
      })
      .finally(() => {
        if (!controller.signal.aborted) loadingProjects = false;
      });
    return () => controller.abort();
  });

  $effect(() => {
    const url = base;
    const selected = projectId;
    void retry;
    connections = [];
    actionError = "";
    if (!activeWorkspace || !canManage || !selected) return;
    const controller = new AbortController();
    loading = true;
    loadError = "";
    request(url, "GET", undefined, controller.signal)
      .then((data) => {
        if (!controller.signal.aborted) connections = data.integrations;
      })
      .catch(() => {
        if (!controller.signal.aborted)
          loadError = "Could not load integrations. Try again.";
      })
      .finally(() => {
        if (!controller.signal.aborted) loading = false;
      });
    return () => controller.abort();
  });

  function edit(provider: string) {
    const existing = connections.find((c) => c.provider === provider);
    secret = "";
    chat = existing?.chat_id || "";
    topic = existing?.topic_id || "";
    originalChat = chat;
    originalTopic = topic;
    actionError = "";
    editing = provider;
  }

  function closeEditor() {
    if (busy) return;
    const focused = document.activeElement as HTMLElement | null;
    if (dirty && !window.confirm("Discard unsaved integration changes?")) {
      void tick().then(() => focused?.focus());
      return;
    }
    editing = "";
    secret = "";
    chat = "";
    topic = "";
    actionError = "";
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (busy) return;
    busy = true;
    actionError = "";
    try {
      const saved = await request(`${base}/${editing}`, "PUT", {
        secret,
        chat_id: chat,
        topic_id: topic.trim(),
      });
      connections = [
        ...connections.filter((c) => c.provider !== editing),
        saved,
      ];
      message = `${names[editing]} integration saved.`;
      editing = "";
      secret = "";
      chat = "";
      topic = "";
    } catch (error) {
      actionError =
        error instanceof Error
          ? error.message
          : "Could not save integration. Try again.";
    } finally {
      busy = false;
    }
  }

  async function test(provider: string) {
    if (busy) return;
    busy = true;
    actionError = "";
    try {
      await request(`${base}/${provider}/test`, "POST");
      message = `${names[provider]} test message sent.`;
    } catch (error) {
      actionError =
        error instanceof Error
          ? error.message
          : "Could not send test. Try again.";
    } finally {
      try {
        connections = (await request(base)).integrations;
      } catch {
        actionError ||=
          "Could not refresh delivery status. Reload integrations to try again.";
      }
      busy = false;
    }
  }

  async function disconnect() {
    if (busy) return;
    busy = true;
    actionError = "";
    try {
      await request(`${base}/${disconnecting}`, "DELETE");
      connections = connections.filter((c) => c.provider !== disconnecting);
      message = `${names[disconnecting]} disconnected.`;
      disconnecting = "";
    } catch (error) {
      actionError =
        error instanceof Error
          ? error.message
          : "Could not disconnect. Try again.";
    } finally {
      busy = false;
    }
  }
</script>

<svelte:window
  onbeforeunload={(event) => {
    if (editing && dirty) event.preventDefault();
  }}
/>

<section class="integrations" aria-labelledby="integration">
  <div class="page-head">
    <div>
      <h2 id="integration" tabindex="-1">Integration</h2>
      <p>Connect a Project to the places your team talks.</p>
    </div>
  </div>
  {#if !signedIn}
    <div class="panel panel-pad">
      <p><a href="#signin">Sign in</a> to manage project integrations.</p>
    </div>
  {:else if !activeWorkspace}
    {#if loadingWorkspace}
      <Feedback tone="neutral" announcement="status"
        >Loading workspace…</Feedback
      >
    {:else}
      <div class="panel panel-pad">
        <p>
          <a href="#workspaces">Choose a Workspace</a> to manage its project integrations.
        </p>
      </div>
    {/if}
  {:else if !canManage}
    <Feedback
      >Only Workspace owners and admins can manage integrations for {activeWorkspace.name}.</Feedback
    >
  {:else}
    {#if loadingProjects || loading}
      <Feedback announcement="status">Loading project integrations…</Feedback>
      <div class="integration-layout" aria-hidden="true">
        <div class="skeleton skeleton-context"></div>
        <div class="provider-stack">
          {#each providers as provider}<div
              class="skeleton skeleton-provider"
            ></div>{/each}
        </div>
      </div>
    {:else if loadError}
      <Feedback tone="danger" announcement="alert">{loadError}</Feedback>
      <Button onclick={() => retry++}>Retry integrations</Button>
    {:else if !projects.length}
      <div class="empty">
        <div class="empty-mark"><Icon name="folder" /></div>
        <h3>No active Projects</h3>
        <p>Create a Project in this Workspace to connect a destination.</p>
        <a href="#workspaces">Open Workspace</a>
      </div>
    {:else}
      <div class="integration-layout">
        <div
          class="panel panel-pad project-context"
          role="group"
          aria-label="Project context"
        >
          <div class="context-heading">
            <span class="tile"><Icon name="folder" size={20} /></span>
            <div>
              <h3>Project connections</h3>
              <p class="note">{activeWorkspace.name}</p>
            </div>
          </div>
          {#if availableWorkspaces.length > 1}
            <div class="field">
              <label for="integration-workspace">Workspace</label>
              <Select
                id="integration-workspace"
                value={activeWorkspace.id}
                options={availableWorkspaces.map((w) => ({
                  value: w.id,
                  label: w.name,
                }))}
                onchange={(id) => {
                  const target = availableWorkspaces.find((w) => w.id === id);
                  if (target) {
                    internalWorkspace = target;
                    onselectworkspace?.(target);
                  }
                }}
                disabled={busy}
              />
            </div>
          {/if}
          <div class="field">
            <label for="integration-project">Project</label>
            <Select
              id="integration-project"
              value={projectId}
              options={projects.map((p) => ({
                value: p.id,
                label: `${p.name} (${p.key})`,
              }))}
              onchange={(id) => (projectId = id)}
              disabled={busy}
              describedby="integration-scope"
            />
          </div>
          <p class="note" id="integration-scope">
            Each connection applies only to this Project.
          </p>
          <div class="context-section">
            <h3>Task events</h3>
            <ul class="event-list">
              {#each ["Task created", "Task updated", "Task deleted"] as event}
                <li><Icon name="check" size={16} />{event}</li>
              {/each}
            </ul>
            <p class="note">
              Updates cover task properties. Comments, labels, and assignees are
              not sent.
            </p>
          </div>
          <div class="context-section">
            <h3>What’s shared</h3>
            <p class="note">
              Project name, task key, title, and Status category. Deletion
              notifications include the task key.
            </p>
          </div>
        </div>
        <div class="provider-stack">
          {#if actionError && !editing && !disconnecting}<Feedback
              tone="danger"
              announcement="alert">{actionError}</Feedback
            >{/if}
          {#each providers as provider}
            {@const connection = connections.find(
              (c) => c.provider === provider,
            )}
            <section
              class="panel panel-pad provider-card"
              aria-labelledby={`provider-${provider}`}
            >
              <div class="provider-identity">
                <span class="tile tile-lg"
                  ><Icon name={provider} size={26} /></span
                >
                <div class="provider-copy">
                  <h3 id={`provider-${provider}`}>{names[provider]}</h3>
                  <p>{descriptions[provider]}</p>
                  <div class="connection-state">
                    <span
                      class="pill"
                      data-tone={connection?.last_error
                        ? "danger"
                        : connection
                          ? "success"
                          : undefined}
                    >
                      <span class="dot" aria-hidden="true"></span>{connection
                        ? "Configured"
                        : "Not connected"}
                    </span>
                    {#if connection?.chat_id}<span class="note"
                        >{connection.chat_id}{#if connection?.topic_id}
                          · Topic {connection.topic_id}{/if}</span
                      >{/if}
                  </div>
                  {#if connection?.last_attempt_at}
                    <p class="note">
                      Last attempt {relativeTime(connection.last_attempt_at)} · {connection.last_error
                        ? "Failed"
                        : "Delivered"}
                    </p>
                  {/if}
                  {#if connection?.last_error}<Feedback tone="danger"
                      >{connection.last_error}</Feedback
                    >{/if}
                </div>
              </div>
              <div class="page-actions">
                <Button disabled={busy} onclick={() => edit(provider)}
                  >{connection
                    ? `Replace ${names[provider]}`
                    : `Connect ${names[provider]}`}</Button
                >
                {#if connection}
                  <Button disabled={busy} onclick={() => test(provider)}
                    >Test {names[provider]}</Button
                  >
                  <Button
                    variant="danger-quiet"
                    disabled={busy}
                    onclick={() => {
                      actionError = "";
                      disconnecting = provider;
                    }}>Disconnect {names[provider]}</Button
                  >
                {/if}
              </div>
            </section>
          {/each}
          <p class="note delivery-note">
            <Icon name="clock" size={16} />Delivery is best effort, with no
            automatic retries. After saving, use Test to check the destination.
          </p>
        </div>
      </div>
    {/if}
  {/if}
</section>

{#if editing}
  <Modal
    title={`${names[editing]} integration`}
    description={`Send task notifications from ${project?.name}. Saving does not send a message; use Test after saving.`}
    onclose={closeEditor}
  >
    <form class="form-grid" onsubmit={save}>
      <p class="note">
        {editing === "telegram"
          ? "Create a bot with BotFather, add it to the destination chat, and allow it to send messages. For a private conversation, start the bot first."
          : editing === "slack"
            ? "In your Slack app, enable Incoming Webhooks and add a webhook for the destination channel."
            : "In your Discord channel settings, open Integrations → Webhooks and copy the webhook URL."}
      </p>
      <p class="note">
        Credentials are stored on this instance and never shown again. To
        replace a connection, enter its credentials again.
      </p>
      <div class="field">
        <label for="integration-secret"
          >{editing === "telegram" ? "Bot token" : "Webhook URL"}</label
        >
        <input
          id="integration-secret"
          type="password"
          bind:value={secret}
          required
          maxlength="512"
          autocomplete="new-password"
          disabled={busy}
          aria-describedby="integration-form-error"
        />
      </div>
      {#if editing === "telegram"}
        <div class="field">
          <label for="integration-chat">Chat ID</label>
          <input
            id="integration-chat"
            bind:value={chat}
            required
            maxlength="33"
            disabled={busy}
            aria-describedby="integration-chat-hint integration-form-error"
          />
          <p id="integration-chat-hint" class="note">
            A numeric chat ID (including the minus sign for groups), or an
            @channel username.
          </p>
        </div>
        <div class="field">
          <label for="integration-topic"
            >Topic ID <span class="field-hint">(optional)</span></label
          >
          <input
            id="integration-topic"
            bind:value={topic}
            maxlength="20"
            disabled={busy}
            placeholder="e.g. 42"
            aria-describedby="integration-topic-hint integration-form-error"
          />
          <p id="integration-topic-hint" class="note">
            For supergroups with topics/forums enabled, enter the topic's thread
            ID.
          </p>
        </div>
      {/if}
      <div id="integration-form-error">
        {#if actionError}<Feedback tone="danger" announcement="alert"
            >{actionError}</Feedback
          >{/if}
      </div>
      {#if dirty}<p class="note">Unsaved changes</p>{/if}
      <div class="dialog-actions">
        <Button disabled={busy} onclick={closeEditor}>Cancel</Button>
        <Button
          type="submit"
          variant="primary"
          disabled={busy || !secret.trim()}
          >{busy ? "Saving…" : "Save integration"}</Button
        >
      </div>
    </form>
  </Modal>
{/if}

{#if disconnecting}
  <Modal
    title={`Disconnect ${names[disconnecting]}?`}
    description={`Stop notifications from ${project?.name} and remove the saved credentials. You can reconnect by entering credentials again. Existing messages will remain.`}
    onclose={() => {
      if (!busy) disconnecting = "";
    }}
  >
    {#if actionError}<Feedback tone="danger" announcement="alert"
        >{actionError}</Feedback
      >{/if}
    <div class="dialog-actions">
      <Button disabled={busy} onclick={() => (disconnecting = "")}
        >Cancel</Button
      >
      <Button variant="danger" disabled={busy} onclick={disconnect}
        >{busy ? "Disconnecting…" : "Disconnect integration"}</Button
      >
    </div>
  </Modal>
{/if}
{#if message}
  <Toast {message} onclear={() => (message = null)} />
{/if}

<style>
  .integrations {
    min-width: 0;
    container-type: inline-size;
  }
  .integration-layout {
    display: grid;
    grid-template-columns: minmax(14rem, 1fr) minmax(0, 3fr);
    gap: var(--space-5);
    align-items: start;
  }
  h3 {
    font-size: var(--text-ui);
  }
  p {
    font-size: var(--text-small);
    overflow-wrap: anywhere;
  }
  .project-context,
  .provider-stack,
  .context-section,
  .provider-copy {
    display: grid;
    gap: var(--space-4);
    min-width: 0;
  }
  .context-heading,
  .provider-identity,
  .connection-state,
  .delivery-note {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .context-section {
    border-top: var(--border-width) solid var(--line);
    padding-top: var(--space-4);
    gap: var(--space-3);
    align-content: start;
  }
  .event-list {
    display: grid;
    gap: var(--space-2);
    list-style: none;
    margin: 0;
    padding: 0;
    font-size: var(--text-small);
    color: var(--text-secondary);
  }
  .event-list li {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }
  .provider-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: var(--space-4);
    min-height: 9rem;
  }
  .provider-identity {
    align-items: flex-start;
    min-width: 0;
    flex: 1 1 18rem;
  }
  .provider-copy {
    gap: var(--space-2);
    grid-template-columns: minmax(0, 1fr);
  }
  .provider-copy h3 {
    font-size: var(--text-section);
  }
  .connection-state {
    flex-wrap: wrap;
    gap: var(--space-2);
  }
  .connection-state .pill {
    white-space: normal;
    line-height: 1.3;
  }
  .delivery-note {
    align-items: flex-start;
    padding-inline: var(--space-1);
    max-width: 65ch;
  }
  .skeleton {
    border-radius: var(--radius-lg);
    background: var(--surface-sunken);
  }
  .skeleton-context {
    min-height: 22rem;
  }
  .skeleton-provider {
    height: 9rem;
  }
  @container (max-width: 52rem) {
    .integration-layout {
      grid-template-columns: minmax(0, 1fr);
    }
    .project-context {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .context-heading,
    .field,
    #integration-scope {
      grid-column: 1 / -1;
    }
  }
  @container (max-width: 40rem) {
    .project-context {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>

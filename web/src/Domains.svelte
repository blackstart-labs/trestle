<script lang="ts">
  import { onMount } from "svelte";
  import Button from "./ui/Button.svelte";
  import Feedback from "./ui/Feedback.svelte";
  import Toast from "./ui/Toast.svelte";

  type Domain = { public_origin: string; listen_address: string };

  let { ondirtychange }: { ondirtychange?: (dirty: boolean) => void } =
    $props();
  let saved = $state<Domain | null>(null);
  let origin = $state("");
  let loading = $state(true);
  let saving = $state(false);
  let denied = $state(false);
  let conflict = $state(false);
  let invalidOrigin = $state(false);
  let error = $state<string | null>(null);
  let success = $state<string | null>(null);
  const dirty = $derived(
    saved !== null && origin.trim() !== saved.public_origin,
  );

  $effect(() => ondirtychange?.(dirty || saving));
  onMount(() => {
    void load();
    return () => ondirtychange?.(false);
  });

  function beforeUnload(event: BeforeUnloadEvent) {
    if (dirty || saving) {
      event.preventDefault();
      event.returnValue = "";
    }
  }

  async function load(keepDraft = false) {
    loading = true;
    error = null;
    invalidOrigin = false;
    try {
      const response = await fetch("/api/v1/instance/domain", {
        cache: "no-store",
      });
      if (response.status === 401 || response.status === 403) {
        denied = true;
        return;
      }
      if (!response.ok) throw new Error();
      saved = (await response.json()) as Domain;
      if (!keepDraft) origin = saved.public_origin;
      conflict = false;
    } catch {
      error = "Could not load the saved public URL. Try again.";
    } finally {
      loading = false;
    }
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (!saved || saving || loading || conflict || !dirty) return;
    saving = true;
    error = null;
    invalidOrigin = false;
    success = null;
    const token = document.cookie
      .split("; ")
      .find((cookie) => cookie.startsWith("trestle-csrf="));
    try {
      const response = await fetch("/api/v1/instance/domain", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": token?.slice("trestle-csrf=".length) ?? "",
        },
        body: JSON.stringify({
          public_origin: origin.trim(),
          previous_origin: saved.public_origin,
        }),
      });
      if (response.status === 401 || response.status === 403) {
        error =
          "Your session or instance access changed. Check your access before trying again.";
        return;
      }
      if (!response.ok) {
        conflict = response.status === 409;
        invalidOrigin = response.status === 400;
        const data = await response.json().catch(() => null);
        error =
          data?.error?.message || "Could not save the public URL. Try again.";
        return;
      }
      saved = (await response.json()) as Domain;
      origin = saved.public_origin;
      success = "Public URL saved.";
    } catch {
      error =
        "Could not save the public URL. Check your connection and try again.";
    } finally {
      saving = false;
    }
  }

  function discard() {
    if (!saved) return;
    origin = saved.public_origin;
    error = null;
    invalidOrigin = false;
    if (conflict) void load();
  }
</script>

<svelte:window onbeforeunload={beforeUnload} />

<section class="panel panel-pad" aria-labelledby="domains">
  <div class="settings-panel-heading">
    <h2 id="domains" tabindex="-1">Domains</h2>
    <p>Set the address used in task notification links.</p>
  </div>

  {#if denied}
    <Feedback>Only an instance operator can manage the public URL.</Feedback>
  {:else if !saved}
    {#if loading}
      <div class="inset domain-skeleton" aria-hidden="true"></div>
      <Feedback announcement="status">Loading domain settings…</Feedback>
    {:else if error}
      <Feedback tone="danger" announcement="alert">{error}</Feedback>
      <Button onclick={() => void load()}>Try again</Button>
    {/if}
  {:else}
    <form
      class="domain-form"
      aria-labelledby="public-origin-label"
      onsubmit={save}
      novalidate
    >
      <div class="field">
        <label id="public-origin-label" for="public-origin">Public URL</label>
        <input
          id="public-origin"
          type="url"
          autocomplete="url"
          bind:value={origin}
          disabled={saving || loading}
          aria-describedby="public-origin-hint"
          aria-invalid={invalidOrigin ? true : undefined}
          aria-errormessage={invalidOrigin ? "public-origin-error" : undefined}
        />
        <p class="note" id="public-origin-hint">
          Use an HTTPS URL with no path. Leave blank to omit links from task
          notifications.
        </p>
      </div>
      {#if error}
        <div id="public-origin-error">
          <Feedback tone="danger" announcement="alert">{error}</Feedback>
        </div>
      {/if}
      <div class="settings-form-actions">
        {#if dirty}
          <Button variant="quiet" disabled={saving || loading} onclick={discard}
            >Discard changes</Button
          >
        {/if}
        {#if conflict}
          <Button disabled={loading} onclick={() => void load(true)}
            >Reload saved URL</Button
          >
        {/if}
        <Button
          type="submit"
          variant="primary"
          disabled={!dirty || saving || loading || conflict}
        >
          {saving ? "Saving…" : "Save public URL"}
        </Button>
      </div>
    </form>
  {/if}
</section>

{#if saved && !denied}
  <section class="panel panel-pad proxy" aria-labelledby="proxy-heading">
    <div class="settings-panel-heading">
      <h2 id="proxy-heading">Reverse proxy setup</h2>
      <p>
        Point DNS to your proxy, enable HTTPS there, and forward requests to
        Trestle.
      </p>
    </div>
    <dl class="proxy-addresses">
      <div class="inset">
        <dt>Trestle listen address</dt>
        <dd><code>{saved.listen_address}</code></dd>
      </div>
      <div class="inset">
        <dt>Docker and Podman host upstream (default)</dt>
        <dd><code>127.0.0.1:3000</code></dd>
      </div>
    </dl>
    <p class="note">Sign in through the new address before saving it here.</p>
  </section>
{/if}

<Toast message={success} onclear={() => (success = null)} />

<style>
  .domain-form {
    display: grid;
    gap: var(--space-4);
  }
  .proxy {
    margin-top: var(--space-5);
    overflow-wrap: anywhere;
  }
  .proxy-addresses {
    display: grid;
    gap: var(--space-2);
    margin: 0 0 var(--space-3);
  }
  .proxy-addresses .inset {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--space-1) var(--space-4);
    padding: var(--space-3);
    font-size: var(--text-small);
  }
  .proxy-addresses dd {
    margin: 0;
  }
  .domain-skeleton {
    min-block-size: var(--control-height);
    margin-bottom: var(--space-3);
  }
</style>

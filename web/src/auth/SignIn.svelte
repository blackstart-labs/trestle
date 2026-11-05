<script lang="ts">
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import BrandMark from "../ui/BrandMark.svelte";

  export interface User {
    id: string;
    username: string;
    display_name: string;
    locale: string;
    timezone: string;
  }

  let {
    onsuccess,
  }: {
    onsuccess: (user: User) => void;
  } = $props();

  let username = $state("");
  let password = $state("");
  let submitting = $state(false);
  let errorMessage = $state<string | null>(null);

  function fillDemo() {
    username = "admin";
    password = "adminpassword123";
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (submitting) return;

    errorMessage = null;
    submitting = true;

    try {
      const response = await fetch("/api/v1/session", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          username: username.trim(),
          password,
        }),
      });

      // Clear password from state immediately after sending.
      password = "";

      if (response.status === 200) {
        const user = (await response.json()) as User;
        onsuccess(user);
      } else if (response.status === 401) {
        errorMessage = "Invalid username or password.";
      } else if (response.status === 429) {
        errorMessage = "Too many attempts. Please try again later.";
      } else {
        errorMessage = "Could not sign in. Please verify your connection.";
      }
    } catch {
      errorMessage = "Could not sign in. Please verify your connection.";
    } finally {
      submitting = false;
    }
  }
</script>

<div class="auth-wrapper">
  <section aria-labelledby="signin-heading" class="auth-card">
    <div class="auth-header">
      <BrandMark size={44} />
      <h2 id="signin-heading" tabindex="-1">Sign in</h2>
      <p>Use your Trestle account to open your workspaces.</p>
    </div>

    <form onsubmit={submit} class="auth-form">
      <InputField
        id="signin-username"
        label="Username"
        autocomplete="username"
        required={true}
        disabled={submitting}
        bind:value={username}
      />

      <InputField
        id="signin-password"
        label="Password"
        type="password"
        autocomplete="current-password"
        required={true}
        disabled={submitting}
        bind:value={password}
      />

      {#if errorMessage}
        <Feedback tone="danger" announcement="alert">
          {errorMessage}
        </Feedback>
      {/if}

      <Button
        type="submit"
        variant="primary"
        disabled={submitting}
        class="submit"
      >
        {#snippet children()}
          {submitting ? "Signing in…" : "Sign in"}
        {/snippet}
      </Button>
    </form>

    <div class="demo-credentials">
      <div class="demo-text">
        <span class="demo-label">Local development login</span>
        <span><code>admin</code> / <code>adminpassword123</code></span>
      </div>
      <button type="button" class="btn-quiet" onclick={fillDemo}>
        Autofill
      </button>
    </div>

    <p class="auth-foot">
      Accounts are provisioned by your instance operator. There is no public
      sign-up.
    </p>
  </section>
</div>

<style>
  .auth-wrapper {
    display: grid;
    place-items: center;
    min-height: calc(100svh - 14rem);
    padding: var(--space-4) 0;
  }

  .auth-card {
    background:
      var(--card-fill) padding-box,
      var(--card-edge) border-box;
    border: var(--border-width) solid transparent;
    width: min(100%, 26rem);
    display: grid;
    gap: var(--space-6);
    padding: clamp(1.5rem, 1rem + 2vw, 2.25rem);
    border-radius: var(--radius-xl);
    box-shadow:
      var(--shadow-card),
      0 30px 80px -40px var(--accent-glow);
  }

  .auth-header {
    display: grid;
    justify-items: start;
    gap: var(--space-3);
  }

  .auth-header :global(.brand-mark) {
    width: 2.75rem;
    height: 2.75rem;
    border-radius: 0.875rem;
    margin-bottom: var(--space-2);
  }

  .auth-header h2 {
    font-size: 1.625rem;
  }

  .auth-form {
    display: grid;
    gap: var(--space-4);
  }

  .auth-form :global(.submit) {
    width: 100%;
    margin-top: var(--space-1);
  }

  .demo-credentials {
    background:
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    border: var(--border-width) solid transparent;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-2) var(--space-2) var(--space-4);
    border-radius: var(--radius);
  }

  .demo-text {
    display: grid;
    gap: 0.125rem;
    min-width: 0;
    font-size: var(--text-small);
    color: var(--text-secondary);
  }

  .demo-label {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-muted);
  }

  .demo-text code {
    color: var(--text);
  }

  .auth-foot {
    font-size: var(--text-xs);
    color: var(--text-muted);
    text-align: center;
  }
</style>

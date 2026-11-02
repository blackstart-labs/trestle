<script lang="ts">
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";

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
      <div class="auth-logo-badge" aria-hidden="true">
        <svg
          width="28"
          height="28"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M4 19L12 4L20 19" />
          <path d="M7 14H17" />
          <circle cx="12" cy="11" r="1.5" />
        </svg>
      </div>
      <p class="eyebrow">Authentication</p>
      <h2 id="signin-heading" tabindex="-1">Sign in</h2>
      <p class="lead">
        Sign in to your Trestle account to access your workspaces.
      </p>
    </div>

    <form onsubmit={submit} novalidate={false} class="auth-form">
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
        <div class="auth-feedback">
          <Feedback tone="danger" announcement="alert">
            {errorMessage}
          </Feedback>
        </div>
      {/if}

      <div class="form-actions">
        <Button type="submit" disabled={submitting}>
          {#snippet children()}
            {submitting ? "Signing in…" : "Sign in"}
          {/snippet}
        </Button>
      </div>
    </form>

    <div class="demo-credentials">
      <div class="demo-text">
        <span class="demo-label">Local development login</span>
        <code>admin</code> / <code>adminpassword123</code>
      </div>
      <button type="button" class="btn-fill" onclick={fillDemo}>
        Autofill
      </button>
    </div>
  </section>
</div>

<style>
  .auth-wrapper {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: calc(80vh - 120px);
    padding: var(--space-4) 0;
  }

  .auth-card {
    max-width: 28rem;
    width: 100%;
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius-lg);
    padding: var(--space-8);
    box-shadow: var(--shadow-modal);
  }

  .auth-header {
    text-align: center;
    margin-bottom: var(--space-6);
  }

  .auth-logo-badge {
    width: 3.5rem;
    height: 3.5rem;
    margin: 0 auto var(--space-4);
    border-radius: var(--radius-md);
    background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
    color: #ffffff;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 8px 16px -4px rgba(79, 70, 229, 0.4);
  }

  .auth-header h2 {
    font-size: 1.75rem;
    margin-bottom: var(--space-2);
  }

  .auth-header .lead {
    font-size: 0.9375rem;
    margin: 0;
  }

  .auth-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .form-actions {
    margin-top: var(--space-3);
  }

  .form-actions :global(button) {
    width: 100%;
    height: 2.875rem;
    font-size: 1rem;
    font-weight: 600;
    background: linear-gradient(135deg, #4f46e5 0%, #4338ca 100%);
    color: #ffffff;
    border: none;
    box-shadow: 0 4px 12px rgba(79, 70, 229, 0.3);
  }

  .form-actions :global(button:hover:not(:disabled)) {
    filter: brightness(1.1);
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(79, 70, 229, 0.4);
  }

  .auth-feedback {
    margin-bottom: var(--space-3);
  }

  .demo-credentials {
    margin-top: var(--space-6);
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius);
    background: var(--surface);
    border: var(--border-width) dashed var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    font-size: 0.8125rem;
  }

  .demo-label {
    display: block;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 0.125rem;
  }

  .demo-credentials code {
    font-family: var(--font-mono);
    color: var(--accent);
    background: var(--selected);
    padding: 0.125rem 0.25rem;
    border-radius: var(--radius-sm);
  }

  .btn-fill {
    font-size: 0.75rem;
    font-weight: 600;
    padding: 0.35rem 0.65rem;
    min-height: 2.75rem;
    border-radius: var(--radius);
    border: var(--border-width) solid var(--border);
    background: var(--surface-raised);
    color: var(--text);
    cursor: pointer;
    flex-shrink: 0;
  }

  .btn-fill:hover {
    background: var(--selected);
  }
</style>

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

<section aria-labelledby="signin-heading" class="auth-card">
  <p class="eyebrow">Authentication</p>
  <h2 id="signin-heading" tabindex="-1">Sign in</h2>
  <p class="lead">Sign in to your Trestle account to access your workspaces.</p>

  <form onsubmit={submit} novalidate={false}>
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

    <Button type="submit" disabled={submitting}>
      {#snippet children()}
        {submitting ? "Signing in…" : "Sign in"}
      {/snippet}
    </Button>
  </form>
</section>

<style>
  .auth-card {
    max-width: 28rem;
  }
  .auth-feedback {
    margin-bottom: var(--space-4);
  }
</style>

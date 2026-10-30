<script lang="ts">
  import { onMount } from "svelte";
  import SelectField from "./ui/SelectField.svelte";
  import Feedback from "./ui/Feedback.svelte";

  type Theme = "system" | "light" | "dark";
  let theme = $state<Theme>("system");
  let storageError = $state(false);
  let saved = $state(false);
  const options = [
    { value: "system", label: "System preference" },
    { value: "light", label: "Light" },
    { value: "dark", label: "Dark" },
  ];

  function isTheme(value: string | null): value is Theme {
    return value === "system" || value === "light" || value === "dark";
  }

  function apply(value: Theme) {
    theme = value;
    document.documentElement.dataset.theme = value;
  }

  onMount(() => {
    try {
      const stored = localStorage.getItem("trestle.theme");
      apply(isTheme(stored) ? stored : "system");
    } catch {
      apply("system");
      storageError = true;
    }
  });

  function change(value: string) {
    if (!isTheme(value)) return;
    apply(value);
    try {
      localStorage.setItem("trestle.theme", value);
      storageError = false;
      saved = true;
    } catch {
      storageError = true;
      saved = false;
    }
  }
</script>

<section aria-labelledby="appearance">
  <p class="eyebrow">Your browser</p>
  <h2 id="appearance" tabindex="-1">Appearance</h2>
  <SelectField
    id="theme"
    feedbackId="theme-feedback"
    label="Theme"
    hint="Applies immediately. Saved only on this browser; system preference follows your device."
    value={theme}
    {options}
    onchange={change}
  />
  <div id="theme-feedback">
    {#if storageError}
      <Feedback tone="warning" announcement="alert"
        >Theme storage is unavailable. Changes apply to this page but may not
        survive a reload.</Feedback
      >
    {:else if saved}
      <Feedback tone="success">Theme saved on this browser.</Feedback>
    {/if}
  </div>
</section>

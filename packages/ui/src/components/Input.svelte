<script lang="ts">
  interface Props {
    label?: string;
    error?: string;
    type?: string;
    placeholder?: string;
    value?: string;
    required?: boolean;
    disabled?: boolean;
    id?: string;
    oninput?: (e: Event) => void;
    onchange?: (e: Event) => void;
  }
  let {
    label,
    error,
    type = 'text',
    placeholder,
    value = $bindable(''),
    required = false,
    disabled = false,
    id,
    oninput,
    onchange,
  }: Props = $props();

  const inputId = id ?? `input-${Math.random().toString(36).slice(2)}`;
</script>

<div class="field" class:field--error={!!error}>
  {#if label}
    <label class="field__label" for={inputId}>
      {label}{#if required}<span aria-hidden="true"> *</span>{/if}
    </label>
  {/if}
  <input
    {type}
    {placeholder}
    {required}
    {disabled}
    id={inputId}
    bind:value
    oninput={oninput}
    onchange={onchange}
    aria-describedby={error ? `${inputId}-error` : undefined}
    aria-invalid={!!error}
    class="field__input"
  />
  {#if error}
    <p id="{inputId}-error" class="field__error" role="alert">{error}</p>
  {/if}
</div>

<style>
  .field { display: flex; flex-direction: column; gap: 0.25rem; }
  .field__label { font-size: var(--text-sm, 0.875rem); font-weight: var(--weight-medium, 500); color: var(--color-text, #1a1611); }
  .field__input {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-md, 0.375rem);
    font-size: var(--text-sm, 0.875rem);
    font-family: var(--font-sans);
    color: var(--color-text, #1a1611);
    background: var(--color-surface, #fff);
    transition: border-color 140ms;
    outline: none;
  }
  .field__input::placeholder { color: var(--color-placeholder, #a89e8b); }
  .field__input:focus {
    border-color: var(--color-primary, #1b3b8f);
    box-shadow: 0 0 0 3px var(--ring-color, rgb(27 59 143 / 0.18));
  }
  .field--error .field__input { border-color: var(--color-danger, #c0392b); }
  .field__error { font-size: var(--text-xs, 0.75rem); color: var(--color-danger-fg, #8c2a20); margin: 0; }
</style>

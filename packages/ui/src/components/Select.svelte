<script lang="ts">
  interface SelectOption { value: string; label: string; }

  interface Props {
    label?: string;
    options: SelectOption[];
    value?: string;
    placeholder?: string;
    required?: boolean;
    disabled?: boolean;
    id?: string;
    onchange?: (e: Event) => void;
  }

  let {
    label,
    options,
    value = $bindable(''),
    placeholder,
    required = false,
    disabled = false,
    id,
    onchange,
  }: Props = $props();

  const selectId = id ?? `select-${Math.random().toString(36).slice(2)}`;
</script>

<div class="field">
  {#if label}
    <label class="field__label" for={selectId}>
      {label}{#if required}<span aria-hidden="true"> *</span>{/if}
    </label>
  {/if}
  <select
    id={selectId}
    {required}
    {disabled}
    bind:value
    onchange={onchange}
    class="field__select"
  >
    {#if placeholder}
      <option value="" disabled selected hidden>{placeholder}</option>
    {/if}
    {#each options as opt}
      <option value={opt.value}>{opt.label}</option>
    {/each}
  </select>
</div>

<style>
  .field { display: flex; flex-direction: column; gap: 0.25rem; }
  .field__label {
    font-size: var(--text-sm, 0.875rem);
    font-weight: 500;
    color: var(--color-text, #0f172a);
  }
  .field__select {
    padding: 0.5rem 2rem 0.5rem 0.75rem;
    border: 1px solid var(--color-border, #e2e8f0);
    border-radius: var(--radius-md, 0.375rem);
    font-size: var(--text-sm, 0.875rem);
    background-color: var(--color-surface, #fff);
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%2364748b' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.5rem center;
    appearance: none;
    cursor: pointer;
    outline: none;
    transition: border-color 150ms;
  }
  .field__select:focus {
    border-color: var(--color-primary, #2563eb);
    box-shadow: 0 0 0 3px var(--ring-color, rgba(37,99,235,0.15));
  }
  .field__select:disabled { opacity: 0.5; cursor: not-allowed; }
</style>

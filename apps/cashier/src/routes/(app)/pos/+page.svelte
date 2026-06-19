<script lang="ts">
  import { getToken } from '$lib/session';
  import { api, ApiError } from '$lib/api';
  import { Button, Input, Badge, Card, LoadingSpinner } from '@pos-stery/ui';

  interface Product { product_id: string; name: string; sku: string; sale_price: number; base_price: number; }
  interface CartItem extends Product { quantity: number; }

  let searchQuery  = $state('');
  let searchResults: Product[] = $state([]);
  let searching    = $state(false);

  let cart: CartItem[] = $state([]);
  let customerId   = $state('');
  let discountAmount = $state(0);

  let submitting   = $state(false);
  let receipt: { sale_id: string; total_amount: number; items_count: number } | null = $state(null);
  let saleError    = $state('');

  async function searchProducts() {
    if (!searchQuery.trim()) { searchResults = []; return; }
    searching = true;
    try {
      const res = await api.get<{ products: Product[] }>(
        `/products?q=${encodeURIComponent(searchQuery)}&limit=10`,
        getToken() ?? undefined,
      );
      searchResults = res.products ?? [];
    } catch { searchResults = []; }
    finally { searching = false; }
  }

  function addToCart(product: Product) {
    const existing = cart.find(i => i.product_id === product.product_id);
    if (existing) {
      cart = cart.map(i => i.product_id === product.product_id ? { ...i, quantity: i.quantity + 1 } : i);
    } else {
      cart = [...cart, { ...product, quantity: 1 }];
    }
    searchQuery = '';
    searchResults = [];
  }

  function updateQty(productId: string, delta: number) {
    cart = cart
      .map(i => i.product_id === productId ? { ...i, quantity: i.quantity + delta } : i)
      .filter(i => i.quantity > 0);
  }

  function removeItem(productId: string) {
    cart = cart.filter(i => i.product_id !== productId);
  }

  const subtotal = $derived(cart.reduce((sum, i) => sum + (i.sale_price || i.base_price) * i.quantity, 0));
  const total    = $derived(Math.max(0, subtotal - discountAmount));

  async function submitSale() {
    if (!cart.length) return;
    submitting = true;
    saleError  = '';
    receipt    = null;
    try {
      receipt = await api.post<typeof receipt>(
        '/sales',
        {
          items: cart.map(i => ({
            product_id: i.product_id,
            quantity:   i.quantity,
            unit_price: i.sale_price || i.base_price,
          })),
          customer_id:     customerId || undefined,
          discount_amount: discountAmount,
        },
        getToken() ?? undefined,
      );
      cart          = [];
      discountAmount = 0;
      customerId    = '';
    } catch (err) {
      saleError = err instanceof ApiError ? err.message : 'Sale failed.';
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head><title>New Sale — POS-Stery Cashier</title></svelte:head>

<div class="pos-layout">
  <!-- Product search panel -->
  <section class="product-panel" aria-label="Product search">
    <h2 class="panel-title">Products</h2>
    <div class="search-wrap">
      <Input
        placeholder="Search by name or SKU…"
        bind:value={searchQuery}
        oninput={searchProducts}
      />
      {#if searching}<LoadingSpinner size="sm" />{/if}
    </div>

    {#if searchResults.length}
      <ul class="search-results" role="listbox" aria-label="Search results">
        {#each searchResults as product (product.product_id)}
          <li role="option" aria-selected="false">
            <button class="product-row" onclick={() => addToCart(product)}>
              <span class="product-row__name">{product.name}</span>
              <span class="product-row__sku">{product.sku}</span>
              <span class="product-row__price">IDR {(product.sale_price || product.base_price).toLocaleString()}</span>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  <!-- Cart panel -->
  <section class="cart-panel" aria-label="Cart">
    <h2 class="panel-title">Cart</h2>

    {#if receipt}
      <div class="receipt-success" role="status">
        <p>Sale completed!</p>
        <p>Sale ID: <code>{receipt.sale_id}</code></p>
        <p>Total: IDR {(receipt.total_amount ?? total).toLocaleString()}</p>
        <Button onclick={() => (receipt = null)}>New Sale</Button>
      </div>
    {:else}
      {#if !cart.length}
        <p class="cart-empty">No items added yet.</p>
      {:else}
        <ul class="cart-list" role="list">
          {#each cart as item (item.product_id)}
            <li class="cart-item">
              <span class="cart-item__name">{item.name}</span>
              <div class="cart-item__controls">
                <button class="qty-btn" onclick={() => updateQty(item.product_id, -1)} aria-label="Decrease quantity">−</button>
                <span class="qty-val">{item.quantity}</span>
                <button class="qty-btn" onclick={() => updateQty(item.product_id, +1)} aria-label="Increase quantity">+</button>
              </div>
              <span class="cart-item__price">IDR {((item.sale_price || item.base_price) * item.quantity).toLocaleString()}</span>
              <button class="cart-item__remove" onclick={() => removeItem(item.product_id)} aria-label="Remove">✕</button>
            </li>
          {/each}
        </ul>

        <div class="cart-footer">
          <div class="cart-field">
            <label for="discount">Discount (IDR)</label>
            <input id="discount" type="number" min="0" bind:value={discountAmount} />
          </div>
          <div class="cart-field">
            <label for="customer">Customer ID (optional)</label>
            <input id="customer" type="text" bind:value={customerId} placeholder="UUID" />
          </div>

          <div class="cart-total">
            <span>Subtotal</span><span>IDR {subtotal.toLocaleString()}</span>
            {#if discountAmount > 0}
              <span>Discount</span><span>- IDR {discountAmount.toLocaleString()}</span>
            {/if}
            <span class="cart-total__label">Total</span>
            <span class="cart-total__amount">IDR {total.toLocaleString()}</span>
          </div>

          {#if saleError}
            <p class="error-msg" role="alert">{saleError}</p>
          {/if}

          <Button onclick={submitSale} loading={submitting} disabled={submitting || !cart.length}>
            Charge IDR {total.toLocaleString()}
          </Button>
        </div>
      {/if}
    {/if}
  </section>
</div>

<style>
  .pos-layout {
    display: grid;
    grid-template-columns: 1fr 380px;
    gap: 1.5rem;
    height: calc(100vh - 100px);
  }
  .panel-title { font-size: 1rem; font-weight: 600; margin-bottom: 1rem; }
  .search-wrap { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.75rem; }

  .search-results {
    list-style: none; margin: 0; padding: 0;
    border: 1px solid var(--color-border); border-radius: 0.5rem; overflow-y: auto; max-height: 400px;
  }
  .product-row {
    display: flex; align-items: center; gap: 0.75rem; padding: 0.75rem 1rem; width: 100%;
    background: none; border: none; cursor: pointer; text-align: left;
    border-bottom: 1px solid var(--color-border);
  }
  .product-row:last-child { border-bottom: none; }
  .product-row:hover { background: var(--color-surface-hover); }
  .product-row__name  { flex: 1; font-weight: 500; }
  .product-row__sku   { color: var(--color-muted); font-size: 0.75rem; }
  .product-row__price { font-weight: 600; color: var(--color-primary); }

  .cart-panel {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 0.5rem;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
  }
  .cart-empty { color: var(--color-muted); text-align: center; padding: 2rem; }

  .cart-list { list-style: none; margin: 0; padding: 0; flex: 1; overflow-y: auto; }
  .cart-item {
    display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem 0;
    border-bottom: 1px solid var(--color-border);
  }
  .cart-item__name { flex: 1; font-size: 0.875rem; }
  .cart-item__controls { display: flex; align-items: center; gap: 0.25rem; }
  .qty-btn {
    width: 1.75rem; height: 1.75rem; border: 1px solid var(--color-border);
    border-radius: 0.25rem; background: none; cursor: pointer; font-size: 1rem; line-height: 1;
  }
  .qty-btn:hover { background: var(--color-surface-hover); }
  .qty-val { width: 1.5rem; text-align: center; font-weight: 600; }
  .cart-item__price { font-weight: 500; font-size: 0.875rem; white-space: nowrap; }
  .cart-item__remove {
    background: none; border: none; color: var(--color-muted); cursor: pointer; font-size: 0.75rem; padding: 0.25rem;
  }
  .cart-item__remove:hover { color: var(--color-danger); }

  .cart-footer { margin-top: 1rem; display: flex; flex-direction: column; gap: 0.75rem; }
  .cart-field  { display: flex; flex-direction: column; gap: 0.25rem; }
  .cart-field label { font-size: 0.75rem; color: var(--color-muted); }
  .cart-field input {
    border: 1px solid var(--color-border); border-radius: 0.375rem; padding: 0.375rem 0.5rem; font-size: 0.875rem;
  }

  .cart-total { display: grid; grid-template-columns: 1fr auto; gap: 0.375rem 1rem; font-size: 0.875rem; }
  .cart-total__label  { font-weight: 700; font-size: 1rem; }
  .cart-total__amount { font-weight: 700; font-size: 1rem; color: var(--color-primary); }

  .error-msg { color: var(--color-danger); font-size: 0.875rem; margin: 0; }

  .receipt-success {
    text-align: center; padding: 2rem;
    display: flex; flex-direction: column; gap: 0.75rem; align-items: center;
  }
  .receipt-success p { margin: 0; }
  .receipt-success code { font-family: monospace; font-size: 0.75rem; }
</style>

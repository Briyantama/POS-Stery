<script lang="ts">
  import { getToken } from '$lib/session';
  import { api, ApiError } from '$lib/api';
  import { Alert, Badge, Button, Input, LoadingSpinner } from '@pos-stery/ui';

  interface Product { product_id: string; name: string; sku: string; sale_price: number; base_price: number; }
  interface CartItem extends Product { quantity: number; }

  /* ── Search ───────────────────────────────────────────────────── */
  let searchQuery   = $state('');
  let searchResults: Product[] = $state([]);
  let searching     = $state(false);

  /* ── Cart ─────────────────────────────────────────────────────── */
  let cart: CartItem[]  = $state([]);
  let customerId        = $state('');
  let discountAmount    = $state(0);

  /* ── Checkout flow: 'cart' | 'tender' | 'receipt' ────────────── */
  type Stage = 'cart' | 'tender' | 'receipt';
  let stage: Stage = $state('cart');

  /* ── Tender step ─────────────────────────────────────────────── */
  type PayMethod = 'Cash' | 'QRIS' | 'Debit';
  let payMethod: PayMethod = $state('Cash');
  let cashTendered  = $state(0);
  let submitting    = $state(false);
  let saleError     = $state('');

  /* ── Receipt ─────────────────────────────────────────────────── */
  let receiptData: { sale_id: string; total_amount: number; items_count: number } | null = $state(null);
  let receiptCart: CartItem[]   = [];
  let receiptDiscount: number   = 0;
  let receiptMethod: PayMethod  = 'Cash';
  let receiptTendered: number   = 0;

  /* ── Derived ─────────────────────────────────────────────────── */
  const subtotal = $derived(cart.reduce((s, i) => s + price(i) * i.quantity, 0));
  const total    = $derived(Math.max(0, subtotal - discountAmount));
  const change   = $derived(payMethod === 'Cash' ? Math.max(0, cashTendered - total) : 0);

  const QUICK_CASH = $derived([
    Math.ceil(total / 10_000) * 10_000,
    Math.ceil(total / 50_000) * 50_000,
    Math.ceil(total / 100_000) * 100_000,
  ].filter((v, i, a) => v > total && a.indexOf(v) === i).slice(0, 3));

  function price(p: Product) { return p.sale_price || p.base_price; }
  function fmtIDR(n: number) { return 'IDR ' + n.toLocaleString('id-ID'); }
  function now() { return new Date().toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' }); }

  /* ── Search ── */
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

  function updateQty(id: string, delta: number) {
    cart = cart
      .map(i => i.product_id === id ? { ...i, quantity: i.quantity + delta } : i)
      .filter(i => i.quantity > 0);
  }
  function removeItem(id: string) { cart = cart.filter(i => i.product_id !== id); }

  /* ── Proceed to tender ── */
  function goToTender() {
    if (!cart.length) return;
    cashTendered = total;
    stage = 'tender';
  }

  /* ── Submit sale ── */
  async function submitSale() {
    submitting = true;
    saleError  = '';
    try {
      const res = await api.post<typeof receiptData>(
        '/sales',
        {
          items: cart.map(i => ({ product_id: i.product_id, quantity: i.quantity, unit_price: price(i) })),
          customer_id:     customerId || undefined,
          discount_amount: discountAmount,
          payment_method:  payMethod,
        },
        getToken() ?? undefined,
      );
      receiptData     = res;
      receiptCart     = [...cart];
      receiptDiscount = discountAmount;
      receiptMethod   = payMethod;
      receiptTendered = cashTendered;
      stage           = 'receipt';
      cart            = [];
      discountAmount  = 0;
      customerId      = '';
    } catch (err) {
      saleError = err instanceof ApiError ? err.message : 'Sale failed. Please try again.';
    } finally {
      submitting = false;
    }
  }

  function newSale() {
    receiptData = null;
    stage = 'cart';
    saleError = '';
    cashTendered = 0;
    payMethod = 'Cash';
  }
</script>

<svelte:head><title>New Sale — POS-Stery Cashier</title></svelte:head>

<div class="pos-layout">

  <!-- ── LEFT: Product search ────────────────────────────────────── -->
  <section class="product-panel" aria-label="Product search">
    <div class="panel-header">
      <h2 class="panel-title">Products</h2>
    </div>
    <div class="panel-body">
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
                <span class="product-row__price">{fmtIDR(price(product))}</span>
              </button>
            </li>
          {/each}
        </ul>
      {:else if !searchQuery}
        <p class="search-hint">Scan barcode or type a product name / SKU.</p>
      {/if}
    </div>
  </section>

  <!-- ── RIGHT: Cart / Tender / Receipt ─────────────────────────── -->
  <section class="cart-panel" aria-label="Cart">

    <!-- CART STAGE ──────────────────────────────────────────────── -->
    {#if stage === 'cart'}
      <div class="panel-header">
        <h2 class="panel-title">Cart</h2>
        {#if cart.length}<Badge variant="info">{cart.length}</Badge>{/if}
      </div>
      <div class="panel-body panel-body--flex">
        {#if !cart.length}
          <p class="cart-empty">Add items from the product list.</p>
        {:else}
          <!-- Receipt-tape cart list -->
          <ul class="receipt-lines" role="list">
            {#each cart as item (item.product_id)}
              <li class="receipt-line">
                <div class="receipt-line__top">
                  <span class="receipt-line__name">{item.name}</span>
                  <button
                    class="receipt-line__remove"
                    onclick={() => removeItem(item.product_id)}
                    aria-label="Remove {item.name}"
                  >✕</button>
                </div>
                <div class="receipt-line__bottom">
                  <div class="qty-stepper">
                    <button class="qty-btn" onclick={() => updateQty(item.product_id, -1)} aria-label="Decrease">−</button>
                    <span class="qty-val" aria-label="Quantity: {item.quantity}">{item.quantity}</span>
                    <button class="qty-btn" onclick={() => updateQty(item.product_id, +1)} aria-label="Increase">+</button>
                  </div>
                  <span class="receipt-line__unit">× {fmtIDR(price(item))}</span>
                  <span class="receipt-line__amount">{fmtIDR(price(item) * item.quantity)}</span>
                </div>
              </li>
            {/each}
          </ul>

          <!-- Totals -->
          <div class="totals">
            <div class="totals-row">
              <span>Subtotal</span>
              <span class="totals-fig">{fmtIDR(subtotal)}</span>
            </div>
            {#if discountAmount > 0}
              <div class="totals-row totals-row--discount">
                <span>Discount</span>
                <span class="totals-fig">− {fmtIDR(discountAmount)}</span>
              </div>
            {/if}
            <div class="totals-row totals-row--total">
              <span>Total</span>
              <span class="totals-fig totals-fig--total">{fmtIDR(total)}</span>
            </div>
          </div>

          <!-- Optional fields -->
          <div class="cart-opts">
            <Input label="Discount (IDR)" type="number" bind:value={discountAmount as unknown as string} />
            <Input label="Customer ID (optional)" bind:value={customerId} placeholder="UUID" />
          </div>

          <Button size="lg" onclick={goToTender} disabled={!cart.length}>
            Charge {fmtIDR(total)}
          </Button>
        {/if}
      </div>

    <!-- TENDER STAGE ────────────────────────────────────────────── -->
    {:else if stage === 'tender'}
      <div class="panel-header">
        <button class="back-btn" onclick={() => stage = 'cart'} aria-label="Back to cart">← Back</button>
        <h2 class="panel-title">Payment</h2>
      </div>
      <div class="panel-body panel-body--flex">
        <!-- Total due -->
        <div class="tender-due">
          <span class="tender-due__label">Total Due</span>
          <span class="tender-due__amount">{fmtIDR(total)}</span>
        </div>

        <!-- Method selector -->
        <div class="method-group" role="group" aria-label="Payment method">
          {#each (['Cash', 'QRIS', 'Debit'] as PayMethod[]) as method}
            <button
              class="method-btn"
              class:method-btn--active={payMethod === method}
              onclick={() => payMethod = method}
            >{method}</button>
          {/each}
        </div>

        <!-- Cash inputs -->
        {#if payMethod === 'Cash'}
          <div class="cash-section">
            <Input
              label="Cash tendered (IDR)"
              type="number"
              bind:value={cashTendered as unknown as string}
            />
            {#if QUICK_CASH.length}
              <div class="quick-cash">
                {#each QUICK_CASH as chip}
                  <button class="quick-chip" onclick={() => (cashTendered = chip)}>
                    {fmtIDR(chip)}
                  </button>
                {/each}
              </div>
            {/if}
            <div class="change-row" class:change-row--due={cashTendered < total && cashTendered > 0}>
              <span class="change-label">Change due</span>
              <span class="change-amount">{fmtIDR(change)}</span>
            </div>
          </div>
        {:else}
          <p class="method-note">{payMethod} — no cash handling required.</p>
        {/if}

        {#if saleError}<Alert>{saleError}</Alert>{/if}

        <Button
          size="lg"
          onclick={submitSale}
          loading={submitting}
          disabled={submitting || (payMethod === 'Cash' && cashTendered < total)}
        >
          {submitting ? 'Processing…' : `Confirm ${payMethod} · ${fmtIDR(total)}`}
        </Button>
      </div>

    <!-- RECEIPT STAGE ───────────────────────────────────────────── -->
    {:else}
      <div class="panel-header">
        <h2 class="panel-title">Receipt</h2>
      </div>
      <div class="panel-body panel-body--flex panel-body--receipt">
        <div class="receipt-card" role="status" aria-live="polite">
          <!-- PAID stamp -->
          <div class="paid-stamp" aria-label="Paid">
            <svg class="paid-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
              <polyline points="22 4 12 14.01 9 11.01"/>
            </svg>
            <span class="paid-word">PAID</span>
          </div>

          <!-- Receipt header -->
          <div class="rct-header">
            <p class="rct-store">POS-Stery</p>
            <p class="rct-date">{now()}</p>
            <p class="rct-ref">{receiptData?.sale_id ?? '—'}</p>
          </div>

          <div class="rct-divider">- - - - - - - - - - - - - - -</div>

          <!-- Line items -->
          <ul class="rct-lines" role="list">
            {#each receiptCart as item}
              <li class="rct-line">
                <span class="rct-line__name">{item.name}</span>
                <span class="rct-line__qty">{item.quantity}×</span>
                <span class="rct-line__amt">{fmtIDR(price(item) * item.quantity)}</span>
              </li>
            {/each}
          </ul>

          <div class="rct-divider">- - - - - - - - - - - - - - -</div>

          <!-- Totals block -->
          <dl class="rct-totals">
            {#if receiptDiscount > 0}
              <div class="rct-tot-row">
                <dt>Discount</dt>
                <dd>− {fmtIDR(receiptDiscount)}</dd>
              </div>
            {/if}
            <div class="rct-tot-row rct-tot-row--bold">
              <dt>Total</dt>
              <dd>{fmtIDR(receiptData?.total_amount ?? 0)}</dd>
            </div>
            <div class="rct-tot-row">
              <dt>{receiptMethod}</dt>
              {#if receiptMethod === 'Cash'}
                <dd>{fmtIDR(receiptTendered)}</dd>
              {:else}
                <dd>—</dd>
              {/if}
            </div>
            {#if receiptMethod === 'Cash'}
              <div class="rct-tot-row rct-tot-row--change">
                <dt>Change</dt>
                <dd>{fmtIDR(Math.max(0, receiptTendered - (receiptData?.total_amount ?? 0)))}</dd>
              </div>
            {/if}
          </dl>

          <div class="rct-divider">- - - - - - - - - - - - - - -</div>
          <p class="rct-thanks">* * * terima kasih * * *</p>
        </div>

        <Button size="lg" onclick={newSale}>New Sale</Button>
      </div>
    {/if}

  </section>
</div>

<style>
  /* ── POS layout: full height, 1fr + 384px ── */
  .pos-layout {
    display: grid;
    grid-template-columns: 1fr 384px;
    height: 100%;
    overflow: hidden;
  }
  @media (max-width: 768px) {
    .pos-layout { grid-template-columns: 1fr; height: auto; overflow: auto; }
  }

  /* ── Shared panel chrome ── */
  .product-panel,
  .cart-panel {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    height: 100%;
  }
  .product-panel { border-right: 1px solid var(--color-border, #e2dbcd); background: var(--color-surface-2, #f4f0e8); }
  .cart-panel    { background: var(--color-surface, #fff); }

  .panel-header {
    display: flex;
    align-items: center;
    gap: var(--space-3, 0.75rem);
    padding: var(--space-4, 1rem) var(--space-5, 1.25rem);
    border-bottom: 1px solid var(--color-border, #e2dbcd);
    flex-shrink: 0;
    background: var(--color-surface, #fff);
  }
  .panel-title {
    font-size: var(--text-sm, 0.875rem);
    font-weight: var(--weight-semibold, 600);
    color: var(--color-text, #1a1611);
    margin: 0;
    text-transform: uppercase;
    letter-spacing: var(--tracking-wide, 0.08em);
    font-size: var(--text-xs, 0.75rem);
  }

  .panel-body {
    padding: var(--space-4, 1rem) var(--space-5, 1.25rem);
    overflow-y: auto;
    flex: 1;
  }
  .panel-body--flex {
    display: flex;
    flex-direction: column;
    gap: var(--space-4, 1rem);
  }
  .panel-body--receipt { overflow-y: auto; }

  /* ── Product search ── */
  .search-wrap { display: flex; align-items: center; gap: var(--space-2, 0.5rem); }
  .search-hint { font-size: var(--text-sm, 0.875rem); color: var(--color-muted, #7a7060); margin: var(--space-4, 1rem) 0 0; }

  .search-results {
    list-style: none; margin: var(--space-3, 0.75rem) 0 0; padding: 0;
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-lg, 0.5rem);
    overflow-y: auto; max-height: 420px;
    background: var(--color-surface, #fff);
  }
  .product-row {
    display: flex; align-items: center; gap: var(--space-3, 0.75rem);
    padding: 0.75rem 1rem; width: 100%;
    background: none; border: none; cursor: pointer; text-align: left;
    border-bottom: 1px solid var(--color-border, #e2dbcd);
    transition: background 140ms;
  }
  .product-row:last-child { border-bottom: none; }
  .product-row:hover { background: var(--color-surface-hover, #eee8dc); }
  .product-row:active { transform: translateY(1px); }
  .product-row:focus-visible { outline: none; box-shadow: inset 0 0 0 2px var(--color-primary, #1b3b8f); }
  .product-row__name  { flex: 1; font-weight: var(--weight-medium, 500); font-size: var(--text-sm, 0.875rem); }
  .product-row__sku   { font-family: var(--font-mono); color: var(--color-muted, #7a7060); font-size: var(--text-xs, 0.75rem); }
  .product-row__price { font-family: var(--font-mono); font-weight: var(--weight-semibold, 600); color: var(--color-primary, #1b3b8f); font-size: var(--text-sm, 0.875rem); white-space: nowrap; font-variant-numeric: tabular-nums; }

  /* ── Cart empty ── */
  .cart-empty { color: var(--color-muted, #7a7060); text-align: center; padding: var(--space-8, 2rem); margin: auto; }

  /* ── Receipt-tape lines ── */
  .receipt-lines { list-style: none; margin: 0; padding: 0; flex: 1; overflow-y: auto; }
  .receipt-line {
    padding: var(--space-3, 0.75rem) 0;
    border-bottom: 1px dashed var(--color-border, #e2dbcd);
  }
  .receipt-line:last-child { border-bottom: none; }
  .receipt-line__top {
    display: flex; align-items: center; justify-content: space-between;
    margin-bottom: var(--space-1, 0.25rem);
  }
  .receipt-line__name { font-size: var(--text-sm, 0.875rem); font-weight: var(--weight-medium, 500); }
  .receipt-line__remove {
    background: none; border: none; color: var(--color-muted, #7a7060); cursor: pointer;
    font-size: var(--text-xs, 0.75rem); padding: 0.125rem 0.25rem; border-radius: var(--radius-sm);
    transition: color 140ms;
  }
  .receipt-line__remove:hover { color: var(--color-danger, #c0392b); }
  .receipt-line__bottom { display: flex; align-items: center; gap: var(--space-3, 0.75rem); }
  .receipt-line__unit   { font-family: var(--font-mono); font-size: var(--text-xs, 0.75rem); color: var(--color-muted, #7a7060); font-variant-numeric: tabular-nums; }
  .receipt-line__amount { font-family: var(--font-mono); font-size: var(--text-sm, 0.875rem); font-weight: var(--weight-semibold, 600); margin-left: auto; font-variant-numeric: tabular-nums; }

  /* Qty stepper */
  .qty-stepper { display: flex; align-items: center; gap: var(--space-1, 0.25rem); }
  .qty-btn {
    width: 1.625rem; height: 1.625rem;
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-sm, 0.25rem);
    background: none; cursor: pointer; font-size: 0.9rem; line-height: 1;
    transition: background 140ms, transform 90ms;
  }
  .qty-btn:hover  { background: var(--color-surface-hover, #eee8dc); }
  .qty-btn:active { transform: translateY(1px); }
  .qty-btn:focus-visible { outline: none; box-shadow: 0 0 0 2px var(--ring-color); }
  .qty-val { width: 1.5rem; text-align: center; font-family: var(--font-mono); font-weight: var(--weight-semibold, 600); font-size: var(--text-sm, 0.875rem); }

  /* ── Totals ── */
  .totals {
    border-top: 2px solid var(--color-border, #e2dbcd);
    padding-top: var(--space-3, 0.75rem);
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .totals-row {
    display: flex;
    justify-content: space-between;
    font-size: var(--text-sm, 0.875rem);
    color: var(--color-muted, #7a7060);
  }
  .totals-row--discount { color: var(--color-success, #2e7d52); }
  .totals-row--total { color: var(--color-text, #1a1611); border-top: 1px solid var(--color-border, #e2dbcd); margin-top: 0.25rem; padding-top: 0.25rem; }
  .totals-fig {
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
    font-weight: var(--weight-semibold, 600);
  }
  .totals-fig--total { font-size: var(--text-base, 1rem); font-weight: var(--weight-bold, 700); color: var(--color-primary, #1b3b8f); }

  /* ── Cart opts ── */
  .cart-opts { display: flex; flex-direction: column; gap: var(--space-3, 0.75rem); }

  /* ── Tender stage ── */
  .back-btn {
    background: none; border: none; cursor: pointer;
    color: var(--color-muted, #7a7060); font-size: var(--text-sm, 0.875rem);
    padding: 0; transition: color 140ms;
  }
  .back-btn:hover { color: var(--color-text, #1a1611); }

  .tender-due {
    display: flex;
    flex-direction: column;
    gap: var(--space-1, 0.25rem);
    padding: var(--space-4, 1rem);
    background: var(--color-primary-light, #e3e8f5);
    border-radius: var(--radius-md, 0.375rem);
  }
  .tender-due__label { font-size: var(--text-xs, 0.75rem); text-transform: uppercase; letter-spacing: var(--tracking-wide, 0.08em); color: var(--color-info-fg, #142e73); font-weight: var(--weight-medium, 500); }
  .tender-due__amount { font-family: var(--font-mono); font-size: var(--text-3xl, 1.875rem); font-weight: var(--weight-black, 800); color: var(--color-primary, #1b3b8f); letter-spacing: var(--tracking-tight, -0.02em); font-variant-numeric: tabular-nums; }

  /* Payment method buttons */
  .method-group { display: flex; gap: var(--space-2, 0.5rem); }
  .method-btn {
    flex: 1;
    padding: var(--space-3, 0.75rem) var(--space-2, 0.5rem);
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-md, 0.375rem);
    background: var(--color-surface, #fff);
    font-size: var(--text-sm, 0.875rem);
    font-weight: var(--weight-medium, 500);
    cursor: pointer;
    color: var(--color-muted, #7a7060);
    transition: border-color 140ms, color 140ms, background 140ms, transform 90ms;
  }
  .method-btn:hover { border-color: var(--color-primary, #1b3b8f); color: var(--color-primary, #1b3b8f); }
  .method-btn:active { transform: translateY(1px); }
  .method-btn--active {
    border-color: var(--color-primary, #1b3b8f);
    background: var(--color-primary-light, #e3e8f5);
    color: var(--color-primary, #1b3b8f);
    font-weight: var(--weight-semibold, 600);
  }

  .cash-section { display: flex; flex-direction: column; gap: var(--space-3, 0.75rem); }

  /* Quick-cash chips */
  .quick-cash { display: flex; gap: var(--space-2, 0.5rem); flex-wrap: wrap; }
  .quick-chip {
    padding: 0.375rem 0.75rem;
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-full, 9999px);
    background: var(--color-surface, #fff);
    font-family: var(--font-mono);
    font-size: var(--text-xs, 0.75rem);
    font-weight: var(--weight-medium, 500);
    cursor: pointer;
    transition: border-color 140ms, background 140ms, transform 90ms;
    font-variant-numeric: tabular-nums;
  }
  .quick-chip:hover  { border-color: var(--color-primary, #1b3b8f); background: var(--color-primary-light, #e3e8f5); }
  .quick-chip:active { transform: translateY(1px); }

  .change-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    padding: var(--space-3, 0.75rem);
    background: var(--color-success-bg, #e1efe5);
    border-radius: var(--radius-md, 0.375rem);
    transition: background 140ms;
  }
  .change-row--due { background: var(--color-warning-bg, #f8e9cb); }
  .change-label  { font-size: var(--text-sm, 0.875rem); color: var(--color-success-fg, #1c5436); font-weight: var(--weight-medium, 500); }
  .change-amount { font-family: var(--font-mono); font-size: var(--text-xl, 1.25rem); font-weight: var(--weight-bold, 700); color: var(--color-success, #2e7d52); font-variant-numeric: tabular-nums; }
  .change-row--due .change-label  { color: var(--color-warning-fg, #7a4b08); }
  .change-row--due .change-amount { color: var(--color-warning, #b5710e); }

  .method-note { font-size: var(--text-sm, 0.875rem); color: var(--color-muted, #7a7060); margin: 0; }

  /* ── Receipt stage ── */
  .receipt-card {
    background: var(--stone-50, #f7f4ee);
    border: 1px solid var(--color-border, #e2dbcd);
    border-radius: var(--radius-md, 0.375rem);
    padding: var(--space-5, 1.25rem);
    font-size: var(--text-sm, 0.875rem);
    position: relative;
  }

  /* PAID stamp overlay */
  .paid-stamp {
    display: flex;
    align-items: center;
    gap: var(--space-2, 0.5rem);
    justify-content: center;
    margin-bottom: var(--space-4, 1rem);
  }
  .paid-icon { width: 1.5rem; height: 1.5rem; color: var(--color-success, #2e7d52); }
  .paid-word {
    font-size: var(--text-xl, 1.25rem);
    font-weight: var(--weight-black, 800);
    color: var(--color-success, #2e7d52);
    letter-spacing: 0.15em;
  }

  .rct-header { text-align: center; margin-bottom: var(--space-3, 0.75rem); }
  .rct-header p { margin: 0.125rem 0; }
  .rct-store { font-weight: var(--weight-bold, 700); font-size: var(--text-base, 1rem); color: var(--color-text, #1a1611); }
  .rct-date  { font-size: var(--text-xs, 0.75rem); color: var(--color-muted, #7a7060); }
  .rct-ref   { font-family: var(--font-mono); font-size: var(--text-xs, 0.75rem); color: var(--color-muted, #7a7060); }

  .rct-divider { text-align: center; color: var(--color-border-strong, #cfc6b5); font-size: var(--text-xs, 0.75rem); letter-spacing: 0.05em; margin: var(--space-3, 0.75rem) 0; }

  .rct-lines { list-style: none; margin: 0; padding: 0; }
  .rct-line {
    display: flex;
    gap: var(--space-2, 0.5rem);
    align-items: baseline;
    padding: 0.25rem 0;
    font-size: var(--text-xs, 0.75rem);
  }
  .rct-line__name { flex: 1; }
  .rct-line__qty  { font-family: var(--font-mono); color: var(--color-muted, #7a7060); white-space: nowrap; }
  .rct-line__amt  { font-family: var(--font-mono); font-weight: var(--weight-semibold, 600); white-space: nowrap; font-variant-numeric: tabular-nums; }

  .rct-totals { margin: 0; }
  .rct-tot-row {
    display: flex;
    justify-content: space-between;
    padding: 0.25rem 0;
    font-size: var(--text-xs, 0.75rem);
  }
  .rct-tot-row dt { color: var(--color-muted, #7a7060); }
  .rct-tot-row dd { font-family: var(--font-mono); margin: 0; font-variant-numeric: tabular-nums; }
  .rct-tot-row--bold { font-size: var(--text-sm, 0.875rem); font-weight: var(--weight-bold, 700); border-top: 1px solid var(--color-border, #e2dbcd); padding-top: 0.5rem; margin-top: 0.25rem; }
  .rct-tot-row--change dd { color: var(--color-success, #2e7d52); font-weight: var(--weight-semibold, 600); }

  .rct-thanks { text-align: center; font-size: var(--text-xs, 0.75rem); color: var(--color-muted, #7a7060); margin: 0; letter-spacing: 0.05em; }
</style>

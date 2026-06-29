package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const validStoreUUID = "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
const anotherStoreUUID = "b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e"
const validTenantUUID = "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f"

// mockStoreValidator is a test double for StoreOwnershipChecker.
type mockStoreValidator struct {
	valid bool
	err   error
}

func (m *mockStoreValidator) StoreExistsInTenant(_ context.Context, _, _ string) (bool, error) {
	return m.valid, m.err
}

func TestStore(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	alwaysValid := &mockStoreValidator{valid: true}
	alwaysInvalid := &mockStoreValidator{valid: false}

	t.Run("cashier with JWT storeID passes through", func(t *testing.T) {
		t.Parallel()
		handler := Store(StoreDeps{Validator: alwaysValid})(next)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := r.Context()
		ctx = withRole(ctx, "cashier")
		ctx = withStoreID(ctx, validStoreUUID)
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("admin with valid X-Store-ID header passes through", func(t *testing.T) {
		t.Parallel()
		handler := Store(StoreDeps{Validator: alwaysValid})(next)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Store-ID", validStoreUUID)
		ctx := r.Context()
		ctx = withRole(ctx, "admin")
		ctx = withTenantID(ctx, validTenantUUID)
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("admin with store not in tenant returns 403", func(t *testing.T) {
		t.Parallel()
		handler := Store(StoreDeps{Validator: alwaysInvalid})(next)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Store-ID", anotherStoreUUID)
		ctx := r.Context()
		ctx = withRole(ctx, "admin")
		ctx = withTenantID(ctx, validTenantUUID)
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("missing storeID returns 403", func(t *testing.T) {
		t.Parallel()
		handler := Store(StoreDeps{Validator: alwaysValid})(next)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(withRole(r.Context(), "admin"))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("invalid UUID storeID returns 403", func(t *testing.T) {
		t.Parallel()
		handler := Store(StoreDeps{Validator: alwaysValid})(next)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Store-ID", "not-a-uuid")
		ctx := r.Context()
		ctx = withRole(ctx, "admin")
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("cashier header store override is ignored JWT value wins", func(t *testing.T) {
		t.Parallel()
		handler := Store(StoreDeps{Validator: alwaysValid})(next)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Store-ID", anotherStoreUUID)
		ctx := r.Context()
		ctx = withRole(ctx, "cashier")
		ctx = withStoreID(ctx, validStoreUUID)
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 (JWT value wins), got %d", w.Code)
		}
	})
}

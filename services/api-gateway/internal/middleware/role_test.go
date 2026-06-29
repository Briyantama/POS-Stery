package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithRole(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allowed role passes through", func(t *testing.T) {
		t.Parallel()
		handler := WithRole("admin", "cashier")(next)

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(withRole(r.Context(), "admin"))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("disallowed role returns 403", func(t *testing.T) {
		t.Parallel()
		handler := WithRole("admin")(next)

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(withRole(r.Context(), "cashier"))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("empty role returns 403", func(t *testing.T) {
		t.Parallel()
		handler := WithRole("admin", "cashier")(next)

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		// no role set in context
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("stock_manager role allowed explicitly", func(t *testing.T) {
		t.Parallel()
		handler := WithRole("admin", "stock_manager")(next)

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(withRole(r.Context(), "stock_manager"))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

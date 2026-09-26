package api

import (
	"net/http"

	"github.com/hirotomasato/paygatemeapp/static"
)

// Router builds the HTTP route table. API routes are registered first,
// then the frontend SPA is served as a catch-all for all other paths.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Admin auth endpoint (public).
	mux.HandleFunc("POST /api/admin/login", s.adminLogin)
	mux.HandleFunc("GET /api/admin/captcha", s.adminCaptcha)

	// Admin endpoints (JWT-protected).
	mux.Handle("GET /api/admin/stores", s.adminAuth(http.HandlerFunc(s.adminListStores)))
	mux.Handle("POST /api/admin/stores", s.adminAuth(http.HandlerFunc(s.adminCreateStore)))
	mux.Handle("PUT /api/admin/stores/{id}", s.adminAuth(http.HandlerFunc(s.adminUpdateStore)))
	mux.Handle("POST /api/admin/stores/{id}/rotate-key", s.adminAuth(http.HandlerFunc(s.adminRotateStoreKey)))
	mux.Handle("DELETE /api/admin/stores/{id}", s.adminAuth(http.HandlerFunc(s.adminDeleteStore)))
	mux.Handle("GET /api/admin/stores/{id}/webhook-logs", s.adminAuth(http.HandlerFunc(s.adminWebhookLogs)))
	mux.Handle("GET /api/admin/transactions", s.adminAuth(http.HandlerFunc(s.adminListTransactions)))
	mux.Handle("GET /api/admin/provider/status", s.adminAuth(http.HandlerFunc(s.adminProviderStatus)))
	mux.Handle("POST /api/admin/provider/otp", s.adminAuth(http.HandlerFunc(s.adminProviderOtp)))
	mux.Handle("POST /api/admin/provider/verify", s.adminAuth(http.HandlerFunc(s.adminProviderVerify)))
	mux.Handle("POST /api/admin/provider/qris", s.adminAuth(http.HandlerFunc(s.adminProviderQris)))
	mux.Handle("POST /api/admin/provider/logout", s.adminAuth(http.HandlerFunc(s.adminProviderLogout)))
	mux.Handle("GET /api/admin/telegram", s.adminAuth(http.HandlerFunc(s.adminGetTelegram)))
	mux.Handle("PUT /api/admin/telegram", s.adminAuth(http.HandlerFunc(s.adminSaveTelegram)))
	mux.Handle("POST /api/admin/telegram/test", s.adminAuth(http.HandlerFunc(s.adminTestTelegram)))

	// Store-facing API (server-key protected).
	mux.Handle("POST /api/v1/transactions", s.storeAuth(http.HandlerFunc(s.createTransaction)))
	mux.Handle("GET /api/v1/transactions/{id}", s.storeAuth(http.HandlerFunc(s.getTransaction)))
	mux.Handle("POST /api/v1/transactions/{id}/cancel", s.storeAuth(http.HandlerFunc(s.cancelTransaction)))

	// Frontend SPA: serve static assets, fallback to index.html for client-side routing.
	mux.Handle("/", static.Frontend())

	return recoverer(mux)
}

// recoverer converts panics into 500s instead of crashing the server.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
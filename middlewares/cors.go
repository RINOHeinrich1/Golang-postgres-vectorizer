package middlewares

import (
	"net/http"
)

// CORSMiddleware autorise toutes les origines (CORS permissif)
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Autorise tous les origines
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Autorise les méthodes standards
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// Autorise les en-têtes standards
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Gère immédiatement les requêtes OPTIONS
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Passe au handler suivant
		next.ServeHTTP(w, r)
	})
}

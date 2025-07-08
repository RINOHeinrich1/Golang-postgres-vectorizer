package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/RINOHeinrich1/postgres-vectorizer/middlewares"
	"github.com/RINOHeinrich1/postgres-vectorizer/utils"
)

func DeleteVectorizedDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// 🔐 Récupérer l'user ID depuis le contexte injecté par le middleware JWT
	ownerID, ok := middlewares.GetUserIDFromContext(r.Context())
	if !ok || ownerID == "" {
		http.Error(w, "Impossible de récupérer l'ID utilisateur à partir du token", http.StatusUnauthorized)
		return
	}

	// Structure attendue dans le body
	var req struct {
		Source string `json:"source"`  // e.g. dbname/tablename
		ConnID string `json:"conn_id"` // id dans postgresql_connexions
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Source == "" || req.ConnID == "" {
		http.Error(w, "source et conn_id sont requis", http.StatusBadRequest)
		return
	}

	// Étape 1 : suppression dans Qdrant
	if err := utils.DeleteFromQdrantByFilter(ownerID, req.Source); err != nil {
		http.Error(w, "Erreur suppression Qdrant: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Étape 2 : suppression dans Supabase (filtrée aussi par owner_id)
	supabaseKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseKey == "" || supabaseURL == "" {
		http.Error(w, "Clé Supabase ou URL manquante dans .env", http.StatusInternalServerError)
		return
	}

	deleteURL := supabaseURL + "/rest/v1/postgresql_connexions?id=eq." + req.ConnID + "&owner_id=eq." + ownerID
	reqDel, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		http.Error(w, "Erreur création requête Supabase", http.StatusInternalServerError)
		return
	}
	reqDel.Header.Set("apikey", supabaseKey)
	reqDel.Header.Set("Authorization", "Bearer "+supabaseKey)
	reqDel.Header.Set("Content-Type", "application/json")

	supabaseResp, err := http.DefaultClient.Do(reqDel)
	if err != nil || supabaseResp.StatusCode >= 300 {
		http.Error(w, "Erreur suppression Supabase", http.StatusInternalServerError)
		return
	}
	defer supabaseResp.Body.Close()

	// Réponse
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Données Qdrant et connexion Supabase supprimées avec succès",
	})
}

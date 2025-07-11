package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/RINOHeinrich1/postgres-vectorizer/utils"
)

type SearchRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"` // optionnel
}

func AskHandler(w http.ResponseWriter, r *http.Request) {
	// Décoder la requête
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requête JSON invalide", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Query == "" {
		http.Error(w, "Le champ 'query' est requis", http.StatusBadRequest)
		return
	}
	if req.TopK == 0 {
		req.TopK = 5 // Valeur par défaut
	}

	// Générer l'embedding
	vector, err := utils.Embed(req.Query)
	if err != nil {
		http.Error(w, "Erreur génération vecteur : "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Recherche dans Qdrant
	results, err := utils.SearchQdrant(vector, req.TopK)
	if err != nil {
		http.Error(w, "Erreur recherche Qdrant : "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Réponse JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/RINOHeinrich1/postgres-vectorizer/middlewares"
	"github.com/RINOHeinrich1/postgres-vectorizer/utils"
)

func DeleteSinglePointHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	ownerID, ok := middlewares.GetUserIDFromContext(r.Context())
	if !ok || ownerID == "" {
		http.Error(w, "Impossible de récupérer l'ID utilisateur", http.StatusUnauthorized)
		return
	}

	var req struct {
		Source string `json:"source"`  // Ex: "db/table"
		DataID string `json:"data_id"` // ID de la donnée
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Source == "" || req.DataID == "" {
		http.Error(w, "Champs 'source' et 'data_id' requis", http.StatusBadRequest)
		return
	}

	err := utils.DeleteDeterministicPoint(req.Source, ownerID, req.DataID)
	if err != nil {
		http.Error(w, "Erreur suppression Qdrant: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Point supprimé avec succès",
	})
}

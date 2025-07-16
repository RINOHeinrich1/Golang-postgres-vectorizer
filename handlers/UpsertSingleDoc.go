package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/RINOHeinrich1/postgres-vectorizer/middlewares"
	"github.com/RINOHeinrich1/postgres-vectorizer/utils"
)

func UpsertSingleDocumentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := middlewares.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Utilisateur non authentifié", http.StatusUnauthorized)
		return
	}

	type RequestBody struct {
		Text       string `json:"text"`
		Source     string `json:"source"`
		DataID     string `json:"data_id"`
		Contextual string `json:"contextual"`
	}

	var req RequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Text == "" || req.Source == "" || req.DataID == "" {
		http.Error(w, "Champs text, source et data_id requis", http.StatusBadRequest)
		return
	}

	if err := utils.SendToQdrant(req.Text, req.Source, userID, req.DataID, "True", req.Contextual); err != nil {
		http.Error(w, "Erreur lors de l’envoi à Qdrant: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Document inséré avec succès",
	})
}

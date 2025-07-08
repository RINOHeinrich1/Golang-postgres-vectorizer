package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RINOHeinrich1/postgres-vectorizer/models"
)

func ConnectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var params models.ConnParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	if params.SSLMode == "" {
		params.SSLMode = "disable"
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		params.Host, params.Port, params.User, params.Password, params.DBName, params.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		http.Error(w, "Erreur ouverture DB: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	db.SetConnMaxLifetime(time.Second * 5)
	if err := db.Ping(); err != nil {
		http.Error(w, "Erreur connexion DB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Connexion réussie !"}`))
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RINOHeinrich1/postgres-vectorizer/models"
)

func GenerateTestDatabaseHandler(w http.ResponseWriter, r *http.Request) {
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

	// Création table produits enrichie
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS produits (
		id SERIAL PRIMARY KEY,
		nom VARCHAR(100) NOT NULL,
		description TEXT,
		categorie VARCHAR(100),
		prix NUMERIC(10,2) NOT NULL,
		stock INT DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createTableSQL); err != nil {
		http.Error(w, "Erreur création table : "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Produits exemples
	examples := []struct {
		Nom         string
		Description string
		Categorie   string
		Prix        float64
		Stock       int
	}{
		{"Chaise", "Chaise en bois confortable", "Mobilier", 49.99, 10},
		{"models.Table", "models.Table en palissandre", "Mobilier", 149.50, 5},
		{"Lampe", "Lampe LED écoénergétique", "Éclairage", 25.00, 20},
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Erreur début transaction : "+err.Error(), http.StatusInternalServerError)
		return
	}

	stmt, err := tx.Prepare(`
		INSERT INTO produits (nom, description, categorie, prix, stock)
		VALUES ($1, $2, $3, $4, $5)
	`)
	if err != nil {
		http.Error(w, "Erreur préparation insert : "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	for _, p := range examples {
		if _, err := stmt.Exec(p.Nom, p.Description, p.Categorie, p.Prix, p.Stock); err != nil {
			tx.Rollback()
			http.Error(w, "Erreur insertion produit : "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Erreur commit transaction : "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Réponse succès
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "models.Table 'produits' créée et exemples insérés avec succès.",
	})
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/RINOHeinrich1/postgres-vectorizer/models"
	"github.com/RINOHeinrich1/postgres-vectorizer/utils"
)

func StaticVectorizerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req models.FormatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.SSLMode == "" {
		req.SSLMode = "disable"
	}
	if req.PageSize <= 0 {
		req.PageSize = 100
	}
	if req.TableName == "" || req.Template == "" {
		http.Error(w, "table_name et template sont obligatoires", http.StatusBadRequest)
		return
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		req.Host, req.Port, req.User, req.Password, req.DBName, req.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		http.Error(w, "Erreur ouverture DB: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		http.Error(w, "Erreur connexion DB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Préparer le template
	tmpl, err := template.New("line").Parse(req.Template)
	if err != nil {
		http.Error(w, "Erreur parsing template: "+err.Error(), http.StatusBadRequest)
		return
	}

	offset := 0
	totalProcessed := 0

	for {
		rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", req.TableName, req.PageSize, offset))
		if err != nil {
			http.Error(w, "Erreur requête SQL: "+err.Error(), http.StatusInternalServerError)
			return
		}

		cols, err := rows.Columns()
		if err != nil {
			rows.Close()
			http.Error(w, "Erreur récupération colonnes: "+err.Error(), http.StatusInternalServerError)
			return
		}

		count := 0

		for rows.Next() {
			values := make([]interface{}, len(cols))
			valuePtrs := make([]interface{}, len(cols))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				rows.Close()
				http.Error(w, "Erreur scan ligne: "+err.Error(), http.StatusInternalServerError)
				return
			}

			data := make(map[string]interface{})
			for i, col := range cols {
				val := values[i]
				if b, ok := val.([]byte); ok {
					data[col] = string(b)
				} else {
					data[col] = val
				}
			}

			var buf strings.Builder
			if err := tmpl.Execute(&buf, data); err != nil {
				rows.Close()
				http.Error(w, "Erreur exécution template: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Envoi à Qdrant
			//pointID := fmt.Sprintf("%s_%d", req.TableName, totalProcessed)
			source := fmt.Sprintf("%s/%s", req.DBName, req.TableName)
			if err := utils.SendToQdrant(buf.String(), source); err != nil {
				rows.Close()
				http.Error(w, "Erreur envoi à Qdrant: "+err.Error(), http.StatusInternalServerError)
				return
			}

			count++
			totalProcessed++
		}

		rows.Close()

		if count < req.PageSize {
			break
		}
		offset += req.PageSize
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Traitement terminé",
		"lignes_traitees": totalProcessed,
	})
}

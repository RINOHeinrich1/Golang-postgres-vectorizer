package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/RINOHeinrich1/postgres-vectorizer/middlewares"

	"github.com/RINOHeinrich1/postgres-vectorizer/models"
	"github.com/RINOHeinrich1/postgres-vectorizer/utils"
)

func StaticVectorizerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := middlewares.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Utilisateur non authentifié", http.StatusUnauthorized)
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
	if req.Template == "" {
		http.Error(w, "template est obligatoire", http.StatusBadRequest)
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

	// Étape 1 : Parse le template pour extraire les chemins .Table.Colonne
	re := regexp.MustCompile(`{{\s*\.([^.]+)\.([^\s}]+)\s*}}`)
	matches := re.FindAllStringSubmatch(req.Template, -1)
	tableColumnMap := make(map[string][]string)
	for _, match := range matches {
		table := match[1]
		column := match[2]
		tableColumnMap[table] = append(tableColumnMap[table], column)
	}

	if len(tableColumnMap) == 0 {
		http.Error(w, "Aucune variable détectée dans le template", http.StatusBadRequest)
		return
	}

	// Étape 2 : Générer la requête SQL avec JOINs
	query, _, err := utils.GenerateSQLWithJoins(db, tableColumnMap, req.PageSize)
	if err != nil {
		http.Error(w, "Erreur génération SQL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	offset := 0
	totalProcessed := 0

	// Préparer le template
	tmpl, err := template.New("line").Parse(req.Template)
	if err != nil {
		http.Error(w, "Erreur parsing template: "+err.Error(), http.StatusBadRequest)
		return
	}

	for {
		fullQuery := fmt.Sprintf("%s OFFSET %d", query, offset)
		rows, err := db.Query(fullQuery)
		if err != nil {
			http.Error(w, "Erreur exécution SQL: "+err.Error(), http.StatusInternalServerError)
			return
		}

		cols, err := rows.Columns()
		if err != nil {
			rows.Close()
			http.Error(w, "Erreur récupération colonnes: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("COLONNES:", cols)
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

			// Mapper vers un objet structuré pour le template
			templateData := make(map[string]map[string]interface{})
			for i, col := range cols {
				parts := strings.SplitN(col, "_", 2)
				if len(parts) != 2 {
					continue
				}
				table := parts[0]
				field := parts[1]

				if _, ok := templateData[table]; !ok {
					templateData[table] = make(map[string]interface{})
				}

				val := values[i]
				if b, ok := val.([]byte); ok {
					templateData[table][field] = string(b)
				} else {
					templateData[table][field] = val
				}
			}
			log.Println("Data template: ", templateData)
			var buf strings.Builder
			if err := tmpl.Execute(&buf, templateData); err != nil {
				rows.Close()
				http.Error(w, "Erreur exécution template: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Récupération de la clé primaire de la table principale
			var mainTable string
			for t := range tableColumnMap {
				mainTable = t
				break
			}
			uniqueId, err := utils.GetUniqueColumn(db, mainTable)
			if err != nil {
				http.Error(w, "Erreur récupération clé primaire: "+err.Error(), http.StatusInternalServerError)
				return
			}
			idValue := templateData[mainTable][uniqueId]
			dataID := fmt.Sprintf("%v", idValue)
			log.Println("dataID", dataID)
			source := fmt.Sprintf("%s", req.DBName)
			if err := utils.SendToQdrant(buf.String(), source, userID, dataID, "False", "False"); err != nil {
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

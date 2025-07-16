package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
)

func RenderTemplateFromDBHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	supabaseKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseKey == "" || supabaseURL == "" {
		http.Error(w, "Variables d'environnement Supabase manquantes", http.StatusInternalServerError)
		return
	}

	var params struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		DBName   string `json:"dbname"`
		SSLMode  string `json:"ssl_mode"`
		Template string `json:"template"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}

	if params.SSLMode == "" {
		params.SSLMode = "disable"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		params.Host, params.Port, params.User, params.Password, params.DBName, params.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		http.Error(w, "Erreur connexion DB: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Extraire les variables du template
	re := regexp.MustCompile(`{{\s*\.([a-zA-Z0-9_]+)\s*}}`)
	matches := re.FindAllStringSubmatch(params.Template, -1)

	varMap := map[string]interface{}{}

	for _, match := range matches {
		varName := match[1]

		// Appel Supabase REST API
		url := fmt.Sprintf("%s/rest/v1/variables?variable_name=eq.%s&select=request", supabaseURL, varName)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			http.Error(w, "Erreur création requête Supabase: "+err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("apikey", supabaseKey)
		req.Header.Set("Authorization", "Bearer "+supabaseKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "Erreur appel Supabase: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, fmt.Sprintf("Variable '%s' introuvable dans Supabase (code %d)", varName, resp.StatusCode), http.StatusNotFound)
			return
		}

		var result []struct {
			Request string `json:"request"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			http.Error(w, "Erreur décodage JSON Supabase: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if len(result) == 0 {
			http.Error(w, "Variable Supabase vide: "+varName, http.StatusNotFound)
			return
		}

		valeurTrimmed := strings.TrimSpace(result[0].Request)
		log.Println("SQL: ", result)

		// Si c'est une requête SQL
		if strings.HasPrefix(strings.ToUpper(valeurTrimmed), "SELECT") {
			rows, err := db.Query(valeurTrimmed)
			if err != nil {
				http.Error(w, "Erreur exécution SQL dans '"+varName+"': "+err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			columns, _ := rows.Columns()
			var resultRows []map[string]interface{}
			for rows.Next() {
				values := make([]interface{}, len(columns))
				valuePtrs := make([]interface{}, len(columns))
				for i := range values {
					valuePtrs[i] = &values[i]
				}
				rows.Scan(valuePtrs...)
				row := map[string]interface{}{}
				for i, col := range columns {
					val := values[i]
					if b, ok := val.([]byte); ok {
						row[col] = string(b)
					} else {
						row[col] = val
					}
				}
				resultRows = append(resultRows, row)
			}
			varMap[varName] = resultRows
		} else {
			// Sinon valeur simple
			varMap[varName] = valeurTrimmed
		}
	}

	// Appliquer le template
	tmpl, err := template.New("tpl").Parse(params.Template)
	if err != nil {
		http.Error(w, "Erreur parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, varMap); err != nil {
		http.Error(w, "Erreur rendu template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(buf.Bytes())
}

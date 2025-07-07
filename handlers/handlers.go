package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/RINOHeinrich1/postgres-vectorizer/models"
	"github.com/RINOHeinrich1/postgres-vectorizer/utils"
)

func GetTablesHandler(w http.ResponseWriter, r *http.Request) {
	// Récupérer les paramètres depuis query params
	host := r.URL.Query().Get("host")
	port := r.URL.Query().Get("port")
	user := r.URL.Query().Get("user")
	password := r.URL.Query().Get("password")
	dbname := r.URL.Query().Get("dbname")
	sslmode := r.URL.Query().Get("sslmode")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		http.Error(w, "Paramètres de connexion manquants", http.StatusBadRequest)
		return
	}

	if sslmode == "" {
		sslmode = "disable"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

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

	// Requête pour récupérer tables et colonnes
	query := `
	SELECT
		c.table_name,
		c.column_name,
		c.data_type,
		c.is_nullable
	FROM
		information_schema.columns c
	JOIN
		information_schema.tables t
	ON
		c.table_name = t.table_name
	WHERE
		t.table_schema = 'public'
		AND t.table_type = 'BASE TABLE'
	ORDER BY
		c.table_name, c.ordinal_position;
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Erreur requête : "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tablesMap := make(map[string][]models.Column)

	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable); err != nil {
			http.Error(w, "Erreur scan : "+err.Error(), http.StatusInternalServerError)
			return
		}
		tablesMap[tableName] = append(tablesMap[tableName], models.Column{
			ColumnName: columnName,
			DataType:   dataType,
			IsNullable: isNullable,
		})
	}

	// Convertir map en slice
	var tables []models.Table
	for tn, cols := range tablesMap {
		tables = append(tables, models.Table{
			TableName: tn,
			Columns:   cols,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tables)
}

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

func FormatRowsHandler(w http.ResponseWriter, r *http.Request) {
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
			pointID := fmt.Sprintf("%s_%d", req.TableName, totalProcessed)
			if err := utils.SendToQdrant(buf.String(), pointID); err != nil {
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

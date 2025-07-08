package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RINOHeinrich1/postgres-vectorizer/models"
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

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
	columnsQuery := `
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

	columnRows, err := db.Query(columnsQuery)
	if err != nil {
		http.Error(w, "Erreur requête colonnes : "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer columnRows.Close()

	tablesMap := make(map[string][]models.Column)

	for columnRows.Next() {
		var tableName, columnName, dataType, isNullable string
		if err := columnRows.Scan(&tableName, &columnName, &dataType, &isNullable); err != nil {
			http.Error(w, "Erreur scan colonnes : "+err.Error(), http.StatusInternalServerError)
			return
		}
		tablesMap[tableName] = append(tablesMap[tableName], models.Column{
			ColumnName: columnName,
			DataType:   dataType,
			IsNullable: isNullable,
		})
	}

	// Relations (clefs étrangères)
	relationsQuery := `
	SELECT
		tc.table_name AS source_table,
		kcu.column_name AS source_column,
		ccu.table_name AS target_table,
		ccu.column_name AS target_column
	FROM
		information_schema.table_constraints AS tc
	JOIN information_schema.key_column_usage AS kcu
		ON tc.constraint_name = kcu.constraint_name
	JOIN information_schema.constraint_column_usage AS ccu
		ON ccu.constraint_name = tc.constraint_name
	WHERE
		tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = 'public';
	`

	relRows, err := db.Query(relationsQuery)
	if err != nil {
		http.Error(w, "Erreur relations : "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer relRows.Close()

	var foreignKeys []models.ForeignKey
	for relRows.Next() {
		var fk models.ForeignKey
		if err := relRows.Scan(&fk.SourceTable, &fk.SourceColumn, &fk.TargetTable, &fk.TargetColumn); err != nil {
			http.Error(w, "Erreur scan relations : "+err.Error(), http.StatusInternalServerError)
			return
		}
		foreignKeys = append(foreignKeys, fk)
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
	json.NewEncoder(w).Encode(struct {
		Tables      []models.Table      `json:"tables"`
		ForeignKeys []models.ForeignKey `json:"foreign_keys"`
	}{
		Tables:      tables,
		ForeignKeys: foreignKeys,
	})
}

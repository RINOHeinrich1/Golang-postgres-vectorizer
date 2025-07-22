package utils

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

type ForeignKeyRelation struct {
	SourceTable  string
	SourceColumn string
	TargetTable  string
	TargetColumn string
}

func getPrimaryKey(db *sql.DB, tableName string) (string, error) {
	query := fmt.Sprintf(`
		SELECT a.attname
		FROM   pg_index i
		JOIN   pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE  i.indrelid = '"%s"'::regclass AND i.indisprimary;
	`, tableName) // ⚠️ attention à l'injection SQL si la tableName est mal contrôlée

	var primaryKey string
	err := db.QueryRow(query).Scan(&primaryKey)
	if err != nil {
		return "", fmt.Errorf("clé primaire introuvable pour %s : %w", tableName, err)
	}
	return primaryKey, nil
}
func GetUniqueColumn(db *sql.DB, tableName string) (string, error) {
	query := fmt.Sprintf(`
        SELECT a.attname
        FROM pg_index i
        JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
        WHERE i.indrelid = '"%s"'::regclass
        AND i.indisunique
        LIMIT 1;
    `, tableName)

	var uniqueCol string
	err := db.QueryRow(query).Scan(&uniqueCol)
	if err != nil {
		return "", fmt.Errorf("colonne unique introuvable pour %s : %w", tableName, err)
	}
	log.Println("COLONNE UNIQUE", uniqueCol)
	return uniqueCol, nil
}
func GetForeignKeys(db *sql.DB) ([]ForeignKeyRelation, error) {
	query := `
		SELECT 
			tc.table_name AS source_table,
			kcu.column_name AS source_column,
			ccu.table_name AS target_table,
			ccu.column_name AS target_column
		FROM 
			information_schema.table_constraints AS tc 
			JOIN information_schema.key_column_usage AS kcu
			  ON tc.constraint_name = kcu.constraint_name
			  AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage AS ccu
			  ON ccu.constraint_name = tc.constraint_name
			  AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []ForeignKeyRelation
	for rows.Next() {
		var rel ForeignKeyRelation
		if err := rows.Scan(&rel.SourceTable, &rel.SourceColumn, &rel.TargetTable, &rel.TargetColumn); err != nil {
			return nil, err
		}
		relations = append(relations, rel)
	}
	return relations, nil
}
func GenerateSQLWithJoins(db *sql.DB, tableColumnMap map[string][]string, limit int) (string, map[string]string, error) {
	var baseTable string
	for t := range tableColumnMap {
		baseTable = t
		break
	}

	aliasMap := make(map[string]string)
	selectParts := []string{}
	joinParts := []string{}

	// Étape 1 : Ajouter les colonnes uniques (si absentes)
	for table, columns := range tableColumnMap {
		uniqueCol, err := GetUniqueColumn(db, table)
		if err != nil {
			return "", nil, fmt.Errorf("erreur récupération colonne unique de %s : %w", table, err)
		}

		// Vérifie si la colonne unique est déjà dans les colonnes demandées
		found := false
		for _, col := range columns {
			if col == uniqueCol {
				found = true
				break
			}
		}
		if !found {
			tableColumnMap[table] = append(tableColumnMap[table], uniqueCol)
		}
	}

	// Étape 2 : Création des alias
	aliasMap[baseTable] = baseTable

	// Étape 3 : Construction du SELECT
	for table, columns := range tableColumnMap {
		alias := aliasMap[table]
		if alias == "" {
			alias = table
			aliasMap[table] = alias
		}
		for _, col := range columns {
			selectParts = append(selectParts, fmt.Sprintf(`"%s"."%s" AS "%s_%s"`, alias, col, table, col))
		}
	}

	// Étape 4 : Récupération des relations
	relations, err := GetForeignKeys(db)
	if err != nil {
		return "", nil, err
	}

	// Étape 5 : Construction des JOIN
	for table := range tableColumnMap {
		if table == baseTable {
			continue
		}
		joinFound := false
		for _, rel := range relations {
			if rel.SourceTable == baseTable && rel.TargetTable == table {
				joinParts = append(joinParts, fmt.Sprintf(
					`JOIN "%s" ON "%s"."%s" = "%s"."%s"`,
					table,
					baseTable, rel.SourceColumn,
					table, rel.TargetColumn,
				))
				joinFound = true
				break
			} else if rel.TargetTable == baseTable && rel.SourceTable == table {
				joinParts = append(joinParts, fmt.Sprintf(
					`JOIN "%s" ON "%s"."%s" = "%s"."%s"`,
					table,
					table, rel.SourceColumn,
					baseTable, rel.TargetColumn,
				))
				joinFound = true
				break
			}
		}
		if !joinFound {
			return "", nil, fmt.Errorf("relation non trouvée entre %s et %s", baseTable, table)
		}
	}

	// Étape 6 : Construction de la requête finale
	query := fmt.Sprintf("SELECT %s FROM \"%s\" %s LIMIT %d",
		strings.Join(selectParts, ", "),
		baseTable,
		strings.Join(joinParts, " "),
		limit,
	)

	return query, aliasMap, nil
}

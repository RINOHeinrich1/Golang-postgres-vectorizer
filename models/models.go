package models

type ConnParams struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"` // optionnel
}

type Column struct {
	ColumnName string `json:"column_name"`
	DataType   string `json:"data_type"`
	IsNullable string `json:"is_nullable"`
}

type Table struct {
	TableName string   `json:"table_name"`
	Columns   []Column `json:"columns"`
}

// Structure requête format rows
type FormatRequest struct {
	ConnParams
	TableName string `json:"table_name"`
	Template  string `json:"template"`
	PageSize  int    `json:"page_size,omitempty"` // optionnel, défaut 100
}

type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"` // ici vecteur vide car tu n'as pas intégré l'embedding
	Payload map[string]interface{} `json:"payload"`
}

type QdrantUpsertRequest struct {
	Points []QdrantPoint `json:"points"`
}

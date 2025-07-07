package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/RINOHeinrich1/postgres-vectorizer/handlers"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func init() {
	// Charger variables d'environnement depuis .env (silencieux si absent)
	_ = godotenv.Load()
}

func main() {
	http.HandleFunc("/connect", handlers.ConnectHandler)
	http.HandleFunc("/generetestdatabase", handlers.GenerateTestDatabaseHandler)
	http.HandleFunc("/tables", handlers.GetTablesHandler)
	http.HandleFunc("/formatrows", handlers.FormatRowsHandler)

	fmt.Println("Serveur lancé sur http://localhost:7777")
	log.Fatal(http.ListenAndServe(":7777", nil))
}

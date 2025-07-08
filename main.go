package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/RINOHeinrich1/postgres-vectorizer/handlers"
	"github.com/RINOHeinrich1/postgres-vectorizer/middlewares"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func init() {
	// Charger variables d'environnement depuis .env (silencieux si absent)
	_ = godotenv.Load()
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/connect", handlers.ConnectHandler)
	mux.HandleFunc("/generetestdatabase", handlers.GenerateTestDatabaseHandler)
	mux.HandleFunc("/tables", handlers.GetTablesHandler)
	mux.HandleFunc("/staticvectorizer", handlers.StaticVectorizerHandler)

	// Appliquer middleware CORS à tout le mux
	handlerWithCORS := middlewares.CORSMiddleware(mux)

	fmt.Println("Serveur lancé sur http://localhost:7777")
	log.Fatal(http.ListenAndServe(":7777", handlerWithCORS))
}

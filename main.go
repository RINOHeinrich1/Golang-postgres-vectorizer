package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/RINOHeinrich1/postgres-vectorizer/handlers"
	"github.com/RINOHeinrich1/postgres-vectorizer/middlewares"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/qdrant/go-client/qdrant"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	// Charger config Qdrant depuis env
	host := os.Getenv("QDRANT_HOST")
	portStr := os.Getenv("QDRANT_PORT")
	apiKey := os.Getenv("QDRANT_API_KEY")
	collection := os.Getenv("QDRANT_COLLECTION")

	if host == "" || portStr == "" || collection == "" {
		log.Fatal("QDRANT_HOST, QDRANT_PORT ou QDRANT_COLLECTION manquantes dans .env")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("QDRANT_PORT invalide : %v", err)
	}

	cfg := qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
		UseTLS: true,
	}
	client, err := qdrant.NewClient(&cfg)
	if err != nil {
		log.Fatalf("Erreur création client Qdrant : %v", err)
	}

	ctx := context.Background()

	exists, err := client.CollectionExists(ctx, collection)
	if err != nil {
		log.Fatalf("Erreur vérification collection : %v", err)
	}

	if exists {
		fmt.Println("✅ La collection existe déjà.")
	} else {
		fmt.Println("ℹ️ La collection n'existe pas. Création en cours...")

		err := client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collection,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     384,                    // Taille du vecteur, selon ton embedder
						Distance: qdrant.Distance_Cosine, // Cosine, Euclidean, Dot
					},
				},
			},
		})

		if err != nil {
			log.Fatalf("❌ Erreur lors de la création de la collection : %v", err)
		}
		fmt.Println("✅ Collection créée avec succès.")
	}

	// Lancement du serveur HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("/connect", handlers.ConnectHandler)
	mux.HandleFunc("/generetestdatabase", handlers.GenerateTestDatabaseHandler)
	mux.HandleFunc("/tables", handlers.GetTablesHandler)
	mux.HandleFunc("/staticvectorizer", handlers.StaticVectorizerHandler)

	handlerWithCORS := middlewares.CORSMiddleware(mux)

	fmt.Println("🚀 Serveur lancé sur http://localhost:7777")
	log.Fatal(http.ListenAndServe(":7777", handlerWithCORS))
}

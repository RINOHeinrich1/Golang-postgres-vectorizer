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
	_ = godotenv.Load()

	// --- Qdrant ---
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
						Size:     384,
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})

		if err != nil {
			log.Fatalf("❌ Erreur lors de la création de la collection : %v", err)
		}
		// Indexer le champ owner_id
		_, err = client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collection,
			FieldName:      "owner_id",
			FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		})
		if err != nil {
			log.Fatalf("Erreur lors de l'indexation du champ owner_id : %v", err)
		}

		// Indexer le champ source
		_, err = client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collection,
			FieldName:      "source",
			FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		})

		// Indexer le champ data_id
		_, err = client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collection,
			FieldName:      "data_id",
			FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		})

		// Indexer le champ template
		_, err = client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collection,
			FieldName:      "template",
			FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		})
		if err != nil {
			log.Fatalf("Erreur lors de l'indexation du champ source : %v", err)
		}

		fmt.Println("✅ Collection créée avec succès.")
	}

	// --- Serveur HTTP ---
	bindAddr := os.Getenv("BIND_ADDR")
	if bindAddr == "" {
		bindAddr = "127.0.0.1"
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "7777"
	}

	address := fmt.Sprintf("%s:%s", bindAddr, serverPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/connect", handlers.ConnectHandler)
	mux.HandleFunc("/generetestdatabase", handlers.GenerateTestDatabaseHandler)
	mux.HandleFunc("/tables", handlers.GetTablesHandler)
	mux.HandleFunc("/staticvectorizer", handlers.StaticVectorizerHandler)
	mux.HandleFunc("/deletevectorizeddata", handlers.DeleteVectorizedDataHandler)
	mux.HandleFunc("/ask", handlers.AskHandler)
	mux.HandleFunc("/execute", handlers.ExecuteSQLHandler)
	mux.HandleFunc("/upsert-single", handlers.UpsertSingleDocumentHandler)
	mux.HandleFunc("/delete", handlers.DeleteSinglePointHandler)
	mux.HandleFunc("/render", handlers.RenderTemplateFromDBHandler)

	protectedHandler := middlewares.CORSMiddleware(middlewares.JWTMiddleware(mux))

	fmt.Printf("🚀 Serveur lancé sur http://%s\n", address)
	log.Fatal(http.ListenAndServe(address, protectedHandler))
}

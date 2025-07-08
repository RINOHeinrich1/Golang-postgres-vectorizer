package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

func SendToQdrant(text, source string, userId string) error {
	host := os.Getenv("QDRANT_HOST")
	portStr := os.Getenv("QDRANT_PORT")
	apiKey := os.Getenv("QDRANT_API_KEY")
	collection := os.Getenv("QDRANT_COLLECTION")

	if host == "" || portStr == "" || collection == "" {
		return fmt.Errorf("env QDRANT_HOST, QDRANT_PORT ou QDRANT_COLLECTION manquantes")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("QDRANT_PORT invalide : %w", err)
	}

	cfg := qdrant.Config{
		Host:   host,
		Port:   int(port),
		APIKey: apiKey,
		UseTLS: true,
	}
	client, err := qdrant.NewClient(&cfg)
	if err != nil {
		return fmt.Errorf("erreur création client Qdrant : %w", err)
	}

	vector, err := Embed(text)
	if err != nil {
		return fmt.Errorf("erreur embedder : %w", err)
	}

	// Générer un UUID string
	id := uuid.New().String()

	point := &qdrant.PointStruct{
		Id:      qdrant.NewIDUUID(id), // utiliser NewIDString pour un UUID
		Vectors: qdrant.NewVectors(vector...),
		Payload: qdrant.NewValueMap(map[string]any{
			"text":     text,
			"source":   source,
			"owner_id": userId,
		}),
	}

	_, err = client.Upsert(context.Background(), &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("échec upsert Qdrant : %w", err)
	}

	return nil
}

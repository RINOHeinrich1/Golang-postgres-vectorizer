package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/RINOHeinrich1/postgres-vectorizer/models"
)

func SendToQdrant(text string, pointID string) error {
	qdrantURL := os.Getenv("QDRANT_URL")
	apiKey := os.Getenv("QDRANT_API_KEY")
	collection := os.Getenv("QDRANT_COLLECTION")

	if qdrantURL == "" || collection == "" {
		return fmt.Errorf("variables d'environnement QDRANT_URL ou QDRANT_COLLECTION manquantes")
	}

	url := fmt.Sprintf("%s/collections/%s/points?wait=true", qdrantURL, collection)
	vector, err := Embed(text)
	if err != nil {
		return fmt.Errorf("erreur embedding: %w", err)
	}
	point := models.QdrantPoint{
		ID:     pointID,
		Vector: vector,
		Payload: map[string]interface{}{
			"text": text,
		},
	}

	upsertReq := models.QdrantUpsertRequest{
		Points: []models.QdrantPoint{point},
	}

	jsonBody, err := json.Marshal(upsertReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("api-key", apiKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant status %d", resp.StatusCode)
	}

	return nil
}

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type EmbedRequest struct {
	Texts []string `json:"texts"`
	Model string   `json:"model"`
}

type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func Embed(text string) ([]float32, error) {
	url := "https://madachat-embedder.hf.space/embed"

	payload := EmbedRequest{
		Texts: []string{text},
		Model: "",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erreur encodage JSON: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erreur requête HTTP embedder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedder status %d: %s", resp.StatusCode, string(body))
	}

	var result EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erreur parsing réponse embedder: %w", err)
	}

	if len(result.Embeddings) == 0 || len(result.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("vecteur vide reçu depuis l'embedder")
	}

	return result.Embeddings[0], nil
}

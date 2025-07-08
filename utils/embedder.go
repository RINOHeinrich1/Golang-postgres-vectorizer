package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type EmbedRequest struct {
	Input string `json:"input"`
}

type EmbedResponse struct {
	Vector []float64 `json:"vector"`
}

func Embed(text string) ([]float64, error) {
	url := "https://madachat-embedder.hf.space/embed"

	payload := EmbedRequest{
		Input: text,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erreur requête embedder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedder status %d: %s", resp.StatusCode, string(body))
	}

	var result EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erreur décodage embedder: %w", err)
	}

	if len(result.Vector) == 0 {
		return nil, fmt.Errorf("vecteur vide reçu")
	}

	return result.Vector, nil
}

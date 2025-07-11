package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/qdrant/go-client/qdrant"
)

type SearchResult struct {
	ID      interface{}
	Score   float32
	Payload map[string]interface{}
}

func convertPayload(payload map[string]*qdrant.Value) map[string]interface{} {
	result := make(map[string]interface{})

	for key, val := range payload {
		switch v := val.Kind.(type) {
		case *qdrant.Value_StringValue:
			result[key] = v.StringValue
		case *qdrant.Value_BoolValue:
			result[key] = v.BoolValue
		case *qdrant.Value_IntegerValue:
			result[key] = v.IntegerValue
		case *qdrant.Value_DoubleValue:
			result[key] = v.DoubleValue
		case *qdrant.Value_ListValue:
			var list []interface{}
			for _, item := range v.ListValue.GetValues() {
				list = append(list, convertPayload(map[string]*qdrant.Value{"_": item})["_"])
			}
			result[key] = list
		default:
			result[key] = nil
		}
	}

	return result
}

func SearchQdrant(queryVector []float32, topK int) ([]SearchResult, error) {
	host := os.Getenv("QDRANT_HOST")
	portStr := os.Getenv("QDRANT_PORT")
	apiKey := os.Getenv("QDRANT_API_KEY")
	collection := os.Getenv("QDRANT_COLLECTION")

	if host == "" || portStr == "" || collection == "" {
		return nil, fmt.Errorf("QDRANT_HOST, QDRANT_PORT ou QDRANT_COLLECTION non définis")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("QDRANT_PORT invalide : %w", err)
	}

	cfg := qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
		UseTLS: true,
	}

	client, err := qdrant.NewClient(&cfg)
	if err != nil {
		return nil, fmt.Errorf("erreur création client Qdrant : %w", err)
	}

	searchParams := &qdrant.SearchPoints{
		CollectionName: collection,
		Vector:         queryVector,
		Limit:          uint64(topK),
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	}

	resp, err := client.GetPointsClient().Search(context.Background(), searchParams)
	if err != nil {
		return nil, fmt.Errorf("échec recherche Qdrant : %w", err)
	}

	var results []SearchResult
	for _, point := range resp.Result {
		payloadMap := convertPayload(point.Payload)

		results = append(results, SearchResult{
			ID:      point.Id,
			Score:   point.Score,
			Payload: payloadMap,
		})
	}

	return results, nil
}

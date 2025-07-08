package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/qdrant/go-client/qdrant"
)

func DeleteFromQdrantByFilter(ownerID, source string) error {
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
		Port:   port,
		APIKey: apiKey,
		UseTLS: true,
	}

	client, err := qdrant.NewClient(&cfg)
	if err != nil {
		return fmt.Errorf("erreur création client Qdrant : %w", err)
	}

	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "owner_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: ownerID,
							},
						},
					},
				},
			},
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "source",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: source,
							},
						},
					},
				},
			},
		},
	}

	_, err = client.Delete(context.Background(), &qdrant.DeletePoints{
		CollectionName: collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: filter,
			},
		},
		Wait: protoBool(true), // helper function to get *bool
	})
	if err != nil {
		return fmt.Errorf("échec suppression Qdrant : %w", err)
	}

	return nil
}

// protoBool is a helper to convert bool to *bool
func protoBool(b bool) *bool {
	return &b
}

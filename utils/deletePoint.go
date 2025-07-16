package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/qdrant/go-client/qdrant"

	"github.com/google/uuid"
)

func DeleteDeterministicPoint(source, ownerID, dataID string) error {
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

	// Génération de l'UUID déterministe
	name := fmt.Sprintf("%s|%s|%s", source, ownerID, dataID)
	namespace := uuid.MustParse("6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	id := uuid.NewSHA1(namespace, []byte(name)).String()

	_, err = client.Delete(context.Background(), &qdrant.DeletePoints{
		CollectionName: collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						qdrant.NewIDUUID(id),
					},
				},
			},
		},
		Wait: protoBool(true),
	})
	if err != nil {
		return fmt.Errorf("échec suppression du point : %w", err)
	}

	return nil
}

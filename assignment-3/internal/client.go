package internal

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Client(ctx context.Context, constr string) (*mongo.Client, error) {

	options := options.Client().ApplyURI(constr)

	client, err := mongo.Connect(options)
	if err != nil {
		return nil, err
	}

	return client, nil
}

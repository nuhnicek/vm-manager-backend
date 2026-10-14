package vlan

import (
	"context"
	"fmt"
	"vm-manager-backend/types"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Counter struct {
	ID    string `bson:"_id"`
	Value int    `bson:"value"`
}

func GetNextVLAN(conn types.Conn) (int, error) {
	coll := conn.Mongo.Database("lab411").Collection("counters")

	filter := bson.M{"_id": "network"}
	update := bson.M{"$inc": bson.M{"value": 1}}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var updatedDoc Counter
	err := coll.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&updatedDoc)
	if err != nil {
		return 0, fmt.Errorf("failed to increment network counter in mongo: %w", err)
	}

	vlanID := updatedDoc.Value + 100

	if vlanID > 4094 {
		return 0, fmt.Errorf("maximum network limit reached (4094)")
	}

	return vlanID, nil
}

package balancer

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func AllocateGlobalVMID(ctx context.Context, mongoClient *mongo.Client) (int, error) {
	if mongoClient == nil {
		return 0, fmt.Errorf("mongo client is nil")
	}

	coll := mongoClient.Database("lab411").Collection("counters")
	filter := bson.M{"_id": "global_vmid"}
	update := bson.M{
		"$inc":         bson.M{"seq": 1},
		"$setOnInsert": bson.M{"created_at": time.Now()},
	}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var doc struct {
		ID  string `bson:"_id"`
		Seq int    `bson:"seq"`
	}

	err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err != nil {
		return 0, fmt.Errorf("failed to allocate global VMID from mongo: %w", err)
	}

	if doc.Seq < 1000 {
		setFilter := bson.M{"_id": "global_vmid"}
		setUpdate := bson.M{"$set": bson.M{"seq": 1000}}
		_, _ = coll.UpdateOne(ctx, setFilter, setUpdate)
		return 1000, nil
	}

	return doc.Seq, nil
}

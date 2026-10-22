package labs

import (
	"context"
	"fmt"
	"time"
	"vm-manager-backend/types"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ListLabTemplates(conn types.Conn) ([]types.LabTemplate, error) {
	coll := conn.Mongo.Database("lab411").Collection("lab_templates")
	cursor, err := coll.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to query lab templates: %w", err)
	}
	defer cursor.Close(context.TODO())

	var templates []types.LabTemplate
	if err := cursor.All(context.TODO(), &templates); err != nil {
		return nil, fmt.Errorf("failed to decode lab templates: %w", err)
	}
	if templates == nil {
		templates = make([]types.LabTemplate, 0)
	}
	return templates, nil
}

func GetLabTemplate(conn types.Conn, id string) (*types.LabTemplate, error) {
	coll := conn.Mongo.Database("lab411").Collection("lab_templates")
	var tpl types.LabTemplate
	err := coll.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&tpl)
	if err != nil {
		return nil, fmt.Errorf("lab template %s not found: %w", id, err)
	}
	return &tpl, nil
}

func CreateLabTemplate(conn types.Conn, tpl types.LabTemplate, creatorUUID string) (*types.LabTemplate, error) {
	if tpl.Name == "" {
		return nil, fmt.Errorf("lab template name cannot be empty")
	}
	if len(tpl.VMs) == 0 {
		return nil, fmt.Errorf("lab template must contain at least one VM")
	}

	if tpl.ID == "" {
		tpl.ID = "lab-tpl-" + uuid.New().String()[:8]
	}
	tpl.CreatedBy = creatorUUID
	tpl.CreatedAt = time.Now()
	tpl.UpdatedAt = time.Now()
	if tpl.Objectives == nil {
		tpl.Objectives = make([]string, 0)
	}
	if tpl.Networks == nil {
		tpl.Networks = make([]types.LabNetworkLink, 0)
	}

	coll := conn.Mongo.Database("lab411").Collection("lab_templates")
	_, err := coll.InsertOne(context.TODO(), tpl)
	if err != nil {
		return nil, fmt.Errorf("failed to insert lab template: %w", err)
	}

	return &tpl, nil
}

func UpdateLabTemplate(conn types.Conn, id string, tpl types.LabTemplate) error {
	coll := conn.Mongo.Database("lab411").Collection("lab_templates")
	tpl.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"name":        tpl.Name,
			"category":    tpl.Category,
			"difficulty":  tpl.Difficulty,
			"description": tpl.Description,
			"objectives":  tpl.Objectives,
			"vms":         tpl.VMs,
			"networks":    tpl.Networks,
			"updated_at":  tpl.UpdatedAt,
		},
	}
	res, err := coll.UpdateOne(context.TODO(), bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update lab template: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("lab template %s not found", id)
	}
	return nil
}

func DeleteLabTemplate(conn types.Conn, id string) error {
	coll := conn.Mongo.Database("lab411").Collection("lab_templates")
	res, err := coll.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete lab template: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("lab template %s not found", id)
	}
	return nil
}

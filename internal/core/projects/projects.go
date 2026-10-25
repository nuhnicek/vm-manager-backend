package projects

import (
	"context"
	"fmt"
	"time"
	"vm-manager-backend/types"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ListProjects(conn types.Conn, userUUID string, isTeacher bool) ([]types.Project, error) {
	coll := conn.Mongo.Database("lab411").Collection("projects")

	filter := bson.M{}
	if !isTeacher && userUUID != "" {
		filter["owner_uuid"] = userUUID
	}

	cursor, err := coll.Find(context.TODO(), filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer cursor.Close(context.TODO())

	var projects []types.Project
	if err := cursor.All(context.TODO(), &projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects: %w", err)
	}
	if projects == nil {
		projects = make([]types.Project, 0)
	}
	for i := range projects {
		if projects[i].VMIDs == nil {
			projects[i].VMIDs = make([]int, 0)
		}
	}
	return projects, nil
}

func GetProject(conn types.Conn, projectID string) (*types.Project, error) {
	coll := conn.Mongo.Database("lab411").Collection("projects")
	var project types.Project
	err := coll.FindOne(context.TODO(), bson.M{"_id": projectID}).Decode(&project)
	if err != nil {
		return nil, fmt.Errorf("project %s not found: %w", projectID, err)
	}
	if project.VMIDs == nil {
		project.VMIDs = make([]int, 0)
	}
	return &project, nil
}

func CreateProject(conn types.Conn, userUUID string, userEmail string, name string, description string, color string) (*types.Project, error) {
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}
	if color == "" {
		color = "#3b82f6"
	}
	coll := conn.Mongo.Database("lab411").Collection("projects")

	project := types.Project{
		ID:          "proj-" + uuid.New().String()[:12],
		Name:        name,
		Description: description,
		Color:       color,
		OwnerUUID:   userUUID,
		OwnerEmail:  userEmail,
		VMIDs:       make([]int, 0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := coll.InsertOne(context.TODO(), project)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}
	return &project, nil
}

func UpdateProject(conn types.Conn, projectID string, userUUID string, isTeacher bool, name string, description string, color string) (*types.Project, error) {
	coll := conn.Mongo.Database("lab411").Collection("projects")

	existing, err := GetProject(conn, projectID)
	if err != nil {
		return nil, err
	}
	if !isTeacher && existing.OwnerUUID != userUUID {
		return nil, fmt.Errorf("unauthorized: project does not belong to user")
	}

	updateFields := bson.M{
		"updated_at": time.Now(),
	}
	if name != "" {
		updateFields["name"] = name
		existing.Name = name
	}
	if description != "" || description == "" {
		updateFields["description"] = description
		existing.Description = description
	}
	if color != "" {
		updateFields["color"] = color
		existing.Color = color
	}

	_, err = coll.UpdateOne(context.TODO(), bson.M{"_id": projectID}, bson.M{"$set": updateFields})
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}
	return existing, nil
}

func DeleteProject(conn types.Conn, projectID string, userUUID string, isTeacher bool) error {
	coll := conn.Mongo.Database("lab411").Collection("projects")

	existing, err := GetProject(conn, projectID)
	if err != nil {
		return err
	}
	if !isTeacher && existing.OwnerUUID != userUUID {
		return fmt.Errorf("unauthorized: project does not belong to user")
	}

	_, err = coll.DeleteOne(context.TODO(), bson.M{"_id": projectID})
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

func AddVMToProject(conn types.Conn, projectID string, vmID int, userUUID string, isTeacher bool) error {
	coll := conn.Mongo.Database("lab411").Collection("projects")

	existing, err := GetProject(conn, projectID)
	if err != nil {
		return err
	}
	if !isTeacher && existing.OwnerUUID != userUUID {
		return fmt.Errorf("unauthorized: project does not belong to user")
	}

	userFilter := bson.M{}
	if !isTeacher && userUUID != "" {
		userFilter["owner_uuid"] = userUUID
	}
	_, _ = coll.UpdateMany(context.TODO(), userFilter, bson.M{"$pull": bson.M{"vm_ids": vmID}})

	_, err = coll.UpdateOne(context.TODO(), bson.M{"_id": projectID}, bson.M{
		"$addToSet": bson.M{"vm_ids": vmID},
		"$set":      bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		return fmt.Errorf("failed to add VM %d to project: %w", vmID, err)
	}
	return nil
}

func RemoveVMFromProject(conn types.Conn, projectID string, vmID int, userUUID string, isTeacher bool) error {
	coll := conn.Mongo.Database("lab411").Collection("projects")

	existing, err := GetProject(conn, projectID)
	if err != nil {
		return err
	}
	if !isTeacher && existing.OwnerUUID != userUUID {
		return fmt.Errorf("unauthorized: project does not belong to user")
	}

	_, err = coll.UpdateOne(context.TODO(), bson.M{"_id": projectID}, bson.M{
		"$pull": bson.M{"vm_ids": vmID},
		"$set":  bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		return fmt.Errorf("failed to remove VM %d from project: %w", vmID, err)
	}
	return nil
}

func AssignVMProject(conn types.Conn, vmID int, projectID string, userUUID string, isTeacher bool) error {
	coll := conn.Mongo.Database("lab411").Collection("projects")

	userFilter := bson.M{}
	if !isTeacher && userUUID != "" {
		userFilter["owner_uuid"] = userUUID
	}
	_, err := coll.UpdateMany(context.TODO(), userFilter, bson.M{
		"$pull": bson.M{"vm_ids": vmID},
		"$set":  bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		return fmt.Errorf("failed to unassign VM from existing projects: %w", err)
	}

	if projectID == "" || projectID == "none" {
		return nil
	}

	targetProj, err := GetProject(conn, projectID)
	if err != nil {
		return err
	}
	if !isTeacher && targetProj.OwnerUUID != userUUID {
		return fmt.Errorf("unauthorized: target project does not belong to user")
	}

	_, err = coll.UpdateOne(context.TODO(), bson.M{"_id": projectID}, bson.M{
		"$addToSet": bson.M{"vm_ids": vmID},
		"$set":      bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		return fmt.Errorf("failed to add VM to project %s: %w", projectID, err)
	}
	return nil
}

func SetProjectVMs(conn types.Conn, projectID string, vmIDs []int, userUUID string, isTeacher bool) error {
	coll := conn.Mongo.Database("lab411").Collection("projects")

	existing, err := GetProject(conn, projectID)
	if err != nil {
		return err
	}
	if !isTeacher && existing.OwnerUUID != userUUID {
		return fmt.Errorf("unauthorized: project does not belong to user")
	}

	if vmIDs == nil {
		vmIDs = make([]int, 0)
	}

	if len(vmIDs) > 0 {
		userFilter := bson.M{
			"_id": bson.M{"$ne": projectID},
		}
		if !isTeacher && userUUID != "" {
			userFilter["owner_uuid"] = userUUID
		}
		_, _ = coll.UpdateMany(context.TODO(), userFilter, bson.M{
			"$pull": bson.M{"vm_ids": bson.M{"$in": vmIDs}},
		})
	}

	_, err = coll.UpdateOne(context.TODO(), bson.M{"_id": projectID}, bson.M{
		"$set": bson.M{
			"vm_ids":     vmIDs,
			"updated_at": time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set project VMs: %w", err)
	}
	return nil
}

func RemoveVMFromAllProjects(conn types.Conn, vmID int) error {
	coll := conn.Mongo.Database("lab411").Collection("projects")
	_, err := coll.UpdateMany(context.TODO(), bson.M{}, bson.M{
		"$pull": bson.M{"vm_ids": vmID},
	})
	return err
}

func GetVMProjectsMap(conn types.Conn, userUUID string, isTeacher bool) (map[int]types.Project, error) {
	projects, err := ListProjects(conn, userUUID, isTeacher)
	if err != nil {
		return nil, err
	}

	res := make(map[int]types.Project)
	for _, p := range projects {
		for _, vmid := range p.VMIDs {
			res[vmid] = p
		}
	}
	return res, nil
}

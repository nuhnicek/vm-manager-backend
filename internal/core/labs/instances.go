package labs

import (
	"context"
	"fmt"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func ListLabInstances(conn types.Conn, userUUID string, isTeacher bool) ([]types.LabInstance, error) {
	coll := conn.Mongo.Database("lab411").Collection("lab_instances")

	filter := bson.M{}
	if !isTeacher && userUUID != "" {
		filter["owner_uuid"] = userUUID
	}

	cursor, err := coll.Find(context.TODO(), filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list lab instances: %w", err)
	}
	defer cursor.Close(context.TODO())

	var instances []types.LabInstance
	if err := cursor.All(context.TODO(), &instances); err != nil {
		return nil, fmt.Errorf("failed to decode lab instances: %w", err)
	}
	if instances == nil {
		instances = make([]types.LabInstance, 0)
	}
	return instances, nil
}

func GetLabInstance(conn types.Conn, instanceID string) (*types.LabInstance, error) {
	coll := conn.Mongo.Database("lab411").Collection("lab_instances")
	var instance types.LabInstance
	err := coll.FindOne(context.TODO(), bson.M{"_id": instanceID}).Decode(&instance)
	if err != nil {
		return nil, fmt.Errorf("lab instance %s not found: %w", instanceID, err)
	}
	return &instance, nil
}

func StartLabInstance(conn types.Conn, instanceID string, userUUID string, isTeacher bool) error {
	instance, err := GetLabInstance(conn, instanceID)
	if err != nil {
		return err
	}
	if !isTeacher && instance.OwnerUUID != userUUID {
		return fmt.Errorf("unauthorized: lab instance does not belong to user")
	}

	authUUID := userUUID
	if isTeacher {
		authUUID = ""
	}

	for _, vmItem := range instance.VMs {
		_ = vms.StartVM(conn, authUUID, vmItem.VMID)
	}

	coll := conn.Mongo.Database("lab411").Collection("lab_instances")
	_, _ = coll.UpdateOne(context.TODO(), bson.M{"_id": instanceID}, bson.M{"$set": bson.M{"status": "running"}})
	return nil
}

func StopLabInstance(conn types.Conn, instanceID string, userUUID string, isTeacher bool) error {
	instance, err := GetLabInstance(conn, instanceID)
	if err != nil {
		return err
	}
	if !isTeacher && instance.OwnerUUID != userUUID {
		return fmt.Errorf("unauthorized: lab instance does not belong to user")
	}

	authUUID := userUUID
	if isTeacher {
		authUUID = ""
	}

	for _, vmItem := range instance.VMs {
		_ = vms.StopVM(conn, authUUID, vmItem.VMID)
	}

	coll := conn.Mongo.Database("lab411").Collection("lab_instances")
	_, _ = coll.UpdateOne(context.TODO(), bson.M{"_id": instanceID}, bson.M{"$set": bson.M{"status": "stopped"}})
	return nil
}

func DeleteLabInstance(conn types.Conn, instanceID string, userUUID string, isTeacher bool) error {
	instance, err := GetLabInstance(conn, instanceID)
	if err != nil {
		return err
	}
	if !isTeacher && instance.OwnerUUID != userUUID {
		return fmt.Errorf("unauthorized: lab instance does not belong to user")
	}

	authUUID := userUUID
	if isTeacher {
		authUUID = ""
	}

	for _, vmItem := range instance.VMs {
		_ = vms.DeleteVM(conn, authUUID, vmItem.VMID)
	}

	coll := conn.Mongo.Database("lab411").Collection("lab_instances")
	_, err = coll.DeleteOne(context.TODO(), bson.M{"_id": instanceID})
	if err != nil {
		return fmt.Errorf("failed to delete lab instance from mongo: %w", err)
	}

	return nil
}

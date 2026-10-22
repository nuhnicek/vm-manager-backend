package auth_test

import (
	"context"
	"testing"
	"vm-manager-backend/internal/core/auth"
	"vm-manager-backend/types"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestUserRoleTeacher(t *testing.T) {
	mongoClient, err := mongo.Connect(options.Client().ApplyURI("mongodb://admin:Cisco123%40@10.0.10.51:27017/?authSource=admin"))
	if err != nil {
		t.Fatalf("mongo connect failed: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	coll := mongoClient.Database("lab411").Collection("users")
	var doc auth.UserDocument
	err = coll.FindOne(context.Background(), bson.M{"uuid": "7c694cfa-66cd-4895-8ba0-676a0b559f59"}).Decode(&doc)
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	if doc.Role != "teacher" {
		t.Errorf("expected role 'teacher', got %q", doc.Role)
	}
	if !doc.IsTeacher {
		t.Errorf("expected is_teacher to be true, got %v", doc.IsTeacher)
	}
}

func TestLoginFlowWithTempUser(t *testing.T) {
	mongoClient, err := mongo.Connect(options.Client().ApplyURI("mongodb://admin:Cisco123%40@10.0.10.51:27017/?authSource=admin"))
	if err != nil {
		t.Fatalf("mongo connect failed: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	rdb := redis.NewClient(&redis.Options{Addr: "10.0.10.55:6379"})
	defer rdb.Close()

	conn := types.Conn{
		Mongo: mongoClient,
		Redis: rdb,
	}

	coll := mongoClient.Database("lab411").Collection("users")
	testUUID := "test-teacher-uuid-1234"
	coll.DeleteOne(context.Background(), bson.M{"uuid": testUUID})

	_, err = coll.InsertOne(context.Background(), bson.M{
		"uuid":       testUUID,
		"username":   "test_teacher",
		"email":      "test_teacher@test.local",
		"password":   "8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918",
		"role":       "teacher",
		"is_teacher": true,
	})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	defer coll.DeleteOne(context.Background(), bson.M{"uuid": testUUID})

	err, token, userDoc := auth.Login("test_teacher", "admin", conn)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Errorf("expected non-empty token")
	}
	if userDoc.Role != "teacher" {
		t.Errorf("expected role 'teacher', got %q", userDoc.Role)
	}
	if !userDoc.IsTeacher {
		t.Errorf("expected is_teacher true, got %v", userDoc.IsTeacher)
	}

	err, sessionUUID, sessionRole := auth.Session(token, conn)
	if err != nil {
		t.Fatalf("session lookup failed: %v", err)
	}
	if sessionUUID != testUUID {
		t.Errorf("expected session uuid %q, got %q", testUUID, sessionUUID)
	}
	if sessionRole != "teacher" {
		t.Errorf("expected session role 'teacher', got %q", sessionRole)
	}
}

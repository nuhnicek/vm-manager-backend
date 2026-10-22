package teacher

import (
	"context"
	"fmt"
	"strings"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/pkg"
	"vm-manager-backend/types"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ListStudents(conn types.Conn) ([]types.TeacherStudentInfo, error) {
	coll := conn.Mongo.Database("lab411").Collection("users")
	cursor, err := coll.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer cursor.Close(context.TODO())

	var users []types.UserDocument
	if err := cursor.All(context.TODO(), &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	labsColl := conn.Mongo.Database("lab411").Collection("lab_instances")

	var studentList []types.TeacherStudentInfo
	for _, u := range users {
		role := u.Role
		if role == "" {
			if u.Email == "p.beunier.st@spseiostrava.cz" || strings.HasPrefix(u.Email, "teacher") || strings.HasPrefix(u.Email, "admin") {
				role = "teacher"
			} else {
				role = "student"
			}
		}

		userVMs, _ := vms.List(conn, u.UUID)
		runningCount := 0
		for _, v := range userVMs {
			if strings.EqualFold(v.Status, "running") {
				runningCount++
			}
		}

		labCount, _ := labsColl.CountDocuments(context.TODO(), bson.M{"owner_uuid": u.UUID})

		studentList = append(studentList, types.TeacherStudentInfo{
			UUID:       u.UUID,
			Email:      u.Email,
			Role:       role,
			VMCount:    len(userVMs),
			RunningVMs: runningCount,
			LabCount:   int(labCount),
		})
	}

	if studentList == nil {
		studentList = make([]types.TeacherStudentInfo, 0)
	}
	return studentList, nil
}

func CreateStudent(conn types.Conn, email string, password string, role string) (*types.UserInfo, error) {
	if email == "" || password == "" {
		return nil, fmt.Errorf("email and password cannot be empty")
	}

	coll := conn.Mongo.Database("lab411").Collection("users")

	count, err := coll.CountDocuments(context.TODO(), bson.M{"email": email})
	if err != nil {
		return nil, fmt.Errorf("database check error: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	if role == "" {
		role = "student"
	}

	newUser := types.UserDocument{
		UUID:     uuid.New().String(),
		Email:    email,
		Password: pkg.SHA256(password),
		Role:     role,
	}

	_, err = coll.InsertOne(context.TODO(), newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new user: %w", err)
	}

	return &types.UserInfo{
		UUID:  newUser.UUID,
		Email: newUser.Email,
		Role:  newUser.Role,
	}, nil
}

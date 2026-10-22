package vms

import (
	"context"
	"time"
	"vm-manager-backend/types"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type StudentInfo struct {
	UUID      string `bson:"uuid" json:"uuid"`
	Username  string `bson:"username" json:"username"`
	Email     string `bson:"email" json:"email"`
	Role      string `bson:"role" json:"role"`
	IsTeacher bool   `bson:"is_teacher" json:"is_teacher"`
}

func ListStudents(conn types.Conn) ([]StudentInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := conn.Mongo.Database("lab411").Collection("users")
	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var students []StudentInfo
	for cursor.Next(ctx) {
		var s StudentInfo
		if err := cursor.Decode(&s); err == nil {
			if s.Role == "" {
				if s.IsTeacher {
					s.Role = "teacher"
				} else {
					s.Role = "student"
				}
			}
			students = append(students, s)
		}
	}
	if students == nil {
		students = make([]StudentInfo, 0)
	}
	return students, nil
}

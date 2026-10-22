package auth

import (
	"context"
	"fmt"
	"math/rand"
	"time"
	"vm-manager-backend/pkg"
	"vm-manager-backend/types"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type UserDocument struct {
	UUID      string `bson:"uuid" json:"uuid"`
	Email     string `bson:"email" json:"email"`
	Username  string `bson:"username,omitempty" json:"username,omitempty"`
	Password  string `bson:"password" json:"-"`
	Role      string `bson:"role,omitempty" json:"role,omitempty"`
	IsTeacher bool   `bson:"is_teacher,omitempty" json:"is_teacher,omitempty"`
}

func Login(identifier string, password string, conn types.Conn) (error, string, UserDocument) {
	var user UserDocument
	hashedPassword := pkg.SHA256(password)
	filter := bson.M{
		"$or": []bson.M{
			{"email": identifier},
			{"username": identifier},
		},
		"password": hashedPassword,
	}

	coll := conn.Mongo.Database("lab411").Collection("users")
	err := coll.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		collUpper := conn.Mongo.Database("lab411").Collection("Users")
		err = collUpper.FindOne(context.TODO(), filter).Decode(&user)
		if err != nil {
			return err, "", user
		}
	}

	role := user.Role
	if role == "" {
		if user.IsTeacher {
			role = "teacher"
		} else {
			role = "student"
		}
	}
	user.Role = role

	token := pkg.SHA256(user.UUID + user.Email + fmt.Sprintf("%s", time.Now()) + fmt.Sprintf("%d", rand.Int()))
	sessionData := user.UUID + "|" + role
	err = conn.Redis.Set(context.Background(), token, sessionData, 1*time.Hour).Err()
	if err != nil {
		return err, "", user
	}
	return nil, token, user
}

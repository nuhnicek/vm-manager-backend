package auth

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"vm-manager-backend/pkg"
	"vm-manager-backend/types"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func GetUserByUUID(conn types.Conn, userUUID string) (*types.UserDocument, error) {
	coll := conn.Mongo.Database("lab411").Collection("users")
	var user types.UserDocument
	err := coll.FindOne(context.TODO(), bson.M{"uuid": userUUID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	if user.Role == "" {
		if user.Email == "p.beunier.st@spseiostrava.cz" || strings.HasPrefix(user.Email, "teacher") || strings.HasPrefix(user.Email, "admin") {
			user.Role = "teacher"
		} else {
			user.Role = "student"
		}
	}
	return &user, nil
}

func Login(email string, password string, conn types.Conn) (error, string, types.UserDocument) {
	var user types.UserDocument
	coll := conn.Mongo.Database("lab411").Collection("users")
	filter := bson.M{
		"email":    email,
		"password": pkg.SHA256(password),
	}
	result := coll.FindOne(context.TODO(), filter)
	err := result.Decode(&user)

	if err != nil {
		return err, "", types.UserDocument{}
	}

	if user.Role == "" {
		if user.Email == "p.beunier.st@spseiostrava.cz" || strings.HasPrefix(user.Email, "teacher") || strings.HasPrefix(user.Email, "admin") {
			user.Role = "teacher"
		} else {
			user.Role = "student"
		}
		_, _ = coll.UpdateOne(context.TODO(), bson.M{"uuid": user.UUID}, bson.M{"$set": bson.M{"role": user.Role}})
	}

	token := pkg.SHA256(user.UUID + user.Email + fmt.Sprintf("%s", time.Now()) + fmt.Sprintf("%d", rand.Int()))
	err = conn.Redis.Set(context.Background(), token, user.UUID, 1*time.Hour).Err()
	if err != nil {
		return err, "", types.UserDocument{}
	}
	return nil, token, user
}

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
	UUID     string `bson:"uuid"`
	Email    string `bson:"email"`
	Password string `bson:"password"`
}

func Login(email string, password string, conn types.Conn) (error, string) {
	var user UserDocument
	coll := conn.Mongo.Database("lab411").Collection("users")
	filter := bson.D{
		{"$and", bson.A{
			bson.D{{"email", email}},
			bson.D{{"password", pkg.SHA256(password)}},
		}},
	}
	result := coll.FindOne(context.TODO(), filter)
	err := result.Decode(&user)

	if err != nil {
		return err, ""
	}

	token := pkg.SHA256(user.UUID + user.Email + fmt.Sprintf("%s", time.Now()) + fmt.Sprintf("%d", rand.Int()))
	err = conn.Redis.Set(context.Background(), token, user.UUID, 1*time.Hour).Err()
	if err != nil {
		return err, ""
	}
	return nil, token
}

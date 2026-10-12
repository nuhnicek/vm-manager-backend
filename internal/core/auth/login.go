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
	uuid     string
	email    string
	password string
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

	token := pkg.SHA256(user.uuid + user.email + fmt.Sprintf("%s", time.Now()) + fmt.Sprintf("%s", rand.Int()))
	fmt.Println(token)
	err = conn.Redis.Set(context.Background(), token, user.email, 1*time.Hour).Err()
	if err != nil {
		return err, ""
	}
	return nil, token
}

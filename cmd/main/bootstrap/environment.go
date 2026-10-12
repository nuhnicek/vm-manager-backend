package bootstrap

import (
	"errors"
	"os"
	"vm-manager-backend/types"

	"github.com/joho/godotenv"
)

func Enviroment() (error, types.Config) {
	godotenv.Load(".env")
	var config types.Config
	mongoUri := os.Getenv("MONGO_URI")
	if mongoUri == "" {
		return errors.New("Mongo not set"), config
	}
	port := os.Getenv("PORT")
	if port == "" {
		return errors.New("Port not set"), config
	}
	redis := os.Getenv("REDIS_URI")
	if redis == "" {
		return errors.New("Redis not set"), config
	}
	config.MongoURI = mongoUri
	config.Port = port
	config.Redis = redis
	return nil, config
}

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
	proxmoxUsername := os.Getenv("PROXMOX_USERNAME")
	if proxmoxUsername == "" {
		return errors.New("PROXMOX_USERNAME not set"), config
	}
	proxmoxPassword := os.Getenv("PROXMOX_PASSWORD")
	if proxmoxPassword == "" {
		return errors.New("PROXMOX_PASSWORD not set"), config
	}
	proxmoxURI := os.Getenv("PROXMOX_URI")
	if proxmoxURI == "" {
		return errors.New("PROXMOX_URI not set"), config
	}
	config.MongoURI = mongoUri
	config.Port = port
	config.Redis = redis
	config.ProxmoxHostURI = proxmoxURI
	config.ProxmoxUsername = proxmoxUsername
	config.ProxmoxPassword = proxmoxPassword

	return nil, config
}

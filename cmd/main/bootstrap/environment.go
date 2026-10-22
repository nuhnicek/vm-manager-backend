package bootstrap

import (
	"errors"
	"os"
	"strings"
	"vm-manager-backend/types"

	"github.com/joho/godotenv"
)

func Enviroment() (error, types.Config) {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../../../.env")
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
		proxmoxURI = os.Getenv("PROXMOX_URIS")
	}
	if proxmoxURI == "" {
		return errors.New("PROXMOX_URI not set"), config
	}

	rawURIs := strings.Split(proxmoxURI, ",")
	var uris []string
	for _, u := range rawURIs {
		trimmed := strings.TrimSpace(u)
		if trimmed != "" {
			uris = append(uris, trimmed)
		}
	}

	config.MongoURI = mongoUri
	config.Port = port
	config.Redis = redis
	config.ProxmoxURIs = uris
	if len(uris) > 0 {
		config.ProxmoxHostURI = uris[0]
	}
	config.ProxmoxUsername = proxmoxUsername
	config.ProxmoxPassword = proxmoxPassword
	config.ProxmoxNode = os.Getenv("PROXMOX_NODE")

	return nil, config
}

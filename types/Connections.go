package types

import (
	"github.com/luthermonson/go-proxmox"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Conn struct {
	Mongo   *mongo.Client
	Redis   *redis.Client
	Proxmox *proxmox.Client
}

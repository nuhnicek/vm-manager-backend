package bootstrap

import (
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func Initialize(router *gin.Engine) (error, types.Config) {
	var conn types.Conn
	err, config := Enviroment()
	if err != nil {
		return err, config
	}
	err, mongo := ConnectMongo(config.MongoURI)
	if err != nil {
		return err, config
	}
	conn.Mongo = mongo
	err, redis := ConnectRedis(config.Redis)
	if err != nil {
		return err, config
	}
	conn.Redis = redis

	uris := config.ProxmoxURIs
	if len(uris) == 0 && config.ProxmoxHostURI != "" {
		uris = []string{config.ProxmoxHostURI}
	}

	pool, err := ConnectProxmoxPool(config.ProxmoxUsername, config.ProxmoxPassword, uris)
	if err != nil {
		return err, config
	}
	conn.Pool = pool
	if len(pool) > 0 {
		conn.Proxmox = pool[0].Client
		conn.ProxmoxNode = pool[0].NodeName
	}
	if config.ProxmoxNode != "" {
		conn.ProxmoxNode = config.ProxmoxNode
	}

	RegisterRouter(router, conn)
	return nil, config
}

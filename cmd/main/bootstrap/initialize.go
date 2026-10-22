package bootstrap

import (
	"vm-manager-backend/internal/core/balancer"
	"vm-manager-backend/internal/core/labs"
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
	proxmox, err := ConnectProxmox(config.ProxmoxUsername, config.ProxmoxPassword, config.ProxmoxHostURI)
	if err != nil {
		return err, config
	}
	conn.Proxmox = proxmox

	initialNode := types.StandaloneNode{
		ID:       "pve-primary",
		HostURI:  config.ProxmoxHostURI,
		Username: config.ProxmoxUsername,
		Password: config.ProxmoxPassword,
		IsActive: true,
		Weight:   100,
	}
	conn.NodePool = balancer.NewPVENodePool(conn.Mongo, initialNode)

	_ = labs.SeedDefaultLabs(conn)

	RegisterRouter(router, conn)
	return nil, config
}

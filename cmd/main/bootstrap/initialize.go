package bootstrap

import (
	"context"
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

	initialNode := types.StandaloneNode{
		ID:       "node-primary",
		Name:     conn.ProxmoxNode,
		HostURI:  config.ProxmoxHostURI,
		Username: config.ProxmoxUsername,
		Password: config.ProxmoxPassword,
		Weight:   100,
		IsActive: true,
	}
	nodePool := balancer.NewPVENodePool(conn.Mongo, initialNode)
	for idx, p := range conn.Pool {
		if idx > 0 {
			_ = nodePool.RegisterNode(context.Background(), types.StandaloneNode{
				ID:       "node-" + p.NodeName,
				Name:     p.NodeName,
				HostURI:  p.HostURI,
				Username: config.ProxmoxUsername,
				Password: config.ProxmoxPassword,
				Weight:   100,
				IsActive: true,
			})
		}
	}
	conn.NodePool = nodePool

	_ = labs.SeedDefaultLabs(conn)

	RegisterRouter(router, conn)
	return nil, config
}

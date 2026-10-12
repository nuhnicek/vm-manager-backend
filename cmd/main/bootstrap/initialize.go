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
	RegisterRouter(router, conn)
	return nil, config
}

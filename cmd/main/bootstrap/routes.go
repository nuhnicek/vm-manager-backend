package bootstrap

import (
	"vm-manager-backend/internal/core/middleware"
	"vm-manager-backend/internal/delivery/handlers/auth"
	"vm-manager-backend/types"

	"vm-manager-backend/internal/delivery/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterRouter(router *gin.Engine, conn types.Conn) {
	router.GET("/ping", middleware.Auth(conn), handlers.Main(conn))
	router.POST("/auth/login", auth.Login(conn))
}

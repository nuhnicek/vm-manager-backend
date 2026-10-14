package bootstrap

import (
	"vm-manager-backend/internal/core/middleware"
	"vm-manager-backend/internal/delivery/handlers/auth"
	"vm-manager-backend/internal/delivery/handlers/vms"
	"vm-manager-backend/internal/delivery/handlers/vms/network"
	"vm-manager-backend/types"

	"vm-manager-backend/internal/delivery/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterRouter(router *gin.Engine, conn types.Conn) {
	router.GET("/ping", middleware.Auth(conn), handlers.Main(conn))
	router.POST("/auth/login", auth.Login(conn))
	router.GET("/vms", middleware.Auth(conn), vms.List(conn))
	router.GET("/vms/templates", middleware.Auth(conn), vms.ListTemplates(conn))
	router.POST("/vms/create", middleware.Auth(conn), vms.CreateFromTemplate(conn))
	router.POST("/vms/start", middleware.Auth(conn), vms.Start(conn))
	router.POST("/vms/stop", middleware.Auth(conn), vms.Stop(conn))
	router.POST("/vms/delete", middleware.Auth(conn), vms.Delete(conn))
	router.POST("/vms/network/create", middleware.Auth(conn), network.CreateNetwork(conn))
	router.POST("/vms/network/disconnect", middleware.Auth(conn), network.DisconnectNetwork(conn))
	router.POST("/vms/network/remove", middleware.Auth(conn), network.RemoveNetwork(conn))
	router.POST("/vms/network/delete", middleware.Auth(conn), network.RemoveNetwork(conn))
	router.POST("/vms/network/add-nic", middleware.Auth(conn), network.AddNIC(conn))
	router.GET("/vms/console/ws", middleware.Auth(conn), vms.ConsoleWS(conn))
	router.GET("/vms/console/vnc", middleware.Auth(conn), vms.VNCWS(conn))
}

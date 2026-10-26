package bootstrap

import (
	"vm-manager-backend/internal/core/middleware"
	"vm-manager-backend/internal/delivery/handlers"
	"vm-manager-backend/internal/delivery/handlers/auth"
	"vm-manager-backend/internal/delivery/handlers/labs"
	"vm-manager-backend/internal/delivery/handlers/projects"
	"vm-manager-backend/internal/delivery/handlers/teacher"
	"vm-manager-backend/internal/delivery/handlers/vms"
	"vm-manager-backend/internal/delivery/handlers/vms/network"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func RegisterRouter(router *gin.Engine, conn types.Conn) {
	router.Use(CORSMiddleware())
	router.GET("/ping", middleware.Auth(conn), handlers.Main(conn))
	router.POST("/auth/login", auth.Login(conn))
	router.GET("/nodes", middleware.Auth(conn), vms.ListNodes(conn))
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
	router.POST("/vms/:id/project", middleware.Auth(conn), projects.AssignVM(conn))
	router.POST("/vms/ip", middleware.Auth(conn), vms.SetIP(conn))
	router.GET("/catalog", middleware.Auth(conn), vms.ListCatalog(conn))
	router.POST("/catalog/deploy", middleware.Auth(conn), vms.DeployCatalog(conn))

	router.GET("/projects", middleware.Auth(conn), projects.List(conn))
	router.POST("/projects", middleware.Auth(conn), projects.Create(conn))
	router.PUT("/projects/:id", middleware.Auth(conn), projects.Update(conn))
	router.DELETE("/projects/:id", middleware.Auth(conn), projects.Delete(conn))
	router.POST("/projects/:id/vms", middleware.Auth(conn), projects.AddVM(conn))
	router.DELETE("/projects/:id/vms/:vmid", middleware.Auth(conn), projects.RemoveVM(conn))

	router.GET("/labs/templates", middleware.Auth(conn), labs.ListTemplates(conn))
	router.GET("/labs/templates/:id", middleware.Auth(conn), labs.GetTemplate(conn))
	router.POST("/labs/templates", middleware.Auth(conn), middleware.RequireTeacher(conn), labs.CreateTemplate(conn))
	router.PUT("/labs/templates/:id", middleware.Auth(conn), middleware.RequireTeacher(conn), labs.UpdateTemplate(conn))
	router.DELETE("/labs/templates/:id", middleware.Auth(conn), middleware.RequireTeacher(conn), labs.DeleteTemplate(conn))

	router.POST("/labs/deploy", middleware.Auth(conn), labs.DeployLab(conn))
	router.GET("/labs/instances", middleware.Auth(conn), labs.ListInstances(conn))
	router.POST("/labs/instances/:id/start", middleware.Auth(conn), labs.StartInstance(conn))
	router.POST("/labs/instances/:id/stop", middleware.Auth(conn), labs.StopInstance(conn))
	router.DELETE("/labs/instances/:id", middleware.Auth(conn), labs.DeleteInstance(conn))

	router.GET("/teacher/students", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.ListStudents(conn))
	router.POST("/teacher/students", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.CreateStudent(conn))
	router.DELETE("/teacher/students/:id", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.DeleteStudent(conn))
	router.POST("/teacher/students/:id/reset-password", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.ResetStudentPassword(conn))
	router.GET("/teacher/vms", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.ListAllVMs(conn))
	router.POST("/teacher/vms/bulk-action", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.BulkAction(conn))

	router.POST("/vms/templates/convert", middleware.Auth(conn), middleware.RequireTeacher(conn), vms.ConvertVMToTemplate(conn))
	router.POST("/vms/templates/clone-as-template", middleware.Auth(conn), middleware.RequireTeacher(conn), vms.CloneVMAsTemplate(conn))

	router.GET("/teacher/nodes", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.ListNodes(conn))
	router.POST("/teacher/nodes", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.RegisterNode(conn))
	router.DELETE("/teacher/nodes/:id", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.RemoveNode(conn))
	router.POST("/teacher/nodes/refresh", middleware.Auth(conn), middleware.RequireTeacher(conn), teacher.RefreshNodes(conn))
}

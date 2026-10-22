package labs

import (
	"fmt"
	"net/http"
	"vm-manager-backend/internal/core/auth"
	"vm-manager-backend/internal/core/labs"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type deployLabBody struct {
	TemplateID  string `json:"template_id"`
	StudentUUID string `json:"student_uuid,omitempty"`
	LabPrefix   string `json:"lab_prefix,omitempty"`
}

func DeployLab(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerUUID, _ := c.Get("uuid")
		callerRole, _ := c.Get("role")
		callerEmail, _ := c.Get("email")

		var body deployLabBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}
		if body.TemplateID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "template_id is required"})
			return
		}

		targetUserUUID := ""
		if callerUUID != nil {
			targetUserUUID = callerUUID.(string)
		}
		targetUserEmail := ""
		if callerEmail != nil {
			targetUserEmail = callerEmail.(string)
		}

		isTeacher := callerRole == "teacher" || callerRole == "admin"
		if isTeacher && body.StudentUUID != "" {
			targetUserUUID = body.StudentUUID
			if studentUser, err := auth.GetUserByUUID(conn, targetUserUUID); err == nil && studentUser != nil {
				targetUserEmail = studentUser.Email
			}
		}

		instance, err := labs.DeployLab(conn, body.TemplateID, targetUserUUID, targetUserEmail, body.LabPrefix)
		if err != nil {
			fmt.Printf("failed to deploy lab: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, instance)
	}
}

func ListInstances(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerUUID, _ := c.Get("uuid")
		callerRole, _ := c.Get("role")

		userUUID := ""
		if callerUUID != nil {
			userUUID = callerUUID.(string)
		}
		isTeacher := callerRole == "teacher" || callerRole == "admin"

		instances, err := labs.ListLabInstances(conn, userUUID, isTeacher)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, instances)
	}
}

func StartInstance(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		callerUUID, _ := c.Get("uuid")
		callerRole, _ := c.Get("role")

		userUUID := ""
		if callerUUID != nil {
			userUUID = callerUUID.(string)
		}
		isTeacher := callerRole == "teacher" || callerRole == "admin"

		err := labs.StartLabInstance(conn, id, userUUID, isTeacher)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "started", "id": id})
	}
}

func StopInstance(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		callerUUID, _ := c.Get("uuid")
		callerRole, _ := c.Get("role")

		userUUID := ""
		if callerUUID != nil {
			userUUID = callerUUID.(string)
		}
		isTeacher := callerRole == "teacher" || callerRole == "admin"

		err := labs.StopLabInstance(conn, id, userUUID, isTeacher)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "stopped", "id": id})
	}
}

func DeleteInstance(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		callerUUID, _ := c.Get("uuid")
		callerRole, _ := c.Get("role")

		userUUID := ""
		if callerUUID != nil {
			userUUID = callerUUID.(string)
		}
		isTeacher := callerRole == "teacher" || callerRole == "admin"

		err := labs.DeleteLabInstance(conn, id, userUUID, isTeacher)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
	}
}

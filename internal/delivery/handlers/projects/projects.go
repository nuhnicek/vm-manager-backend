package projects

import (
	"net/http"
	"strconv"
	"vm-manager-backend/internal/core/projects"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type createProjectBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type updateProjectBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type addVMBody struct {
	VMID  int   `json:"vm_id"`
	VMIDs []int `json:"vm_ids"`
}

type assignVMBody struct {
	ProjectID string `json:"project_id"`
}

func List(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuidVal, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		uuid, _ := uuidVal.(string)
		role, _ := roleVal.(string)
		isTeacher := role == "teacher" || role == "admin"

		list, err := projects.ListProjects(conn, uuid, isTeacher)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, list)
	}
}

func Create(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createProjectBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		uuidVal, _ := c.Get("uuid")
		uuid, _ := uuidVal.(string)
		emailVal, _ := c.Get("email")
		email, _ := emailVal.(string)

		proj, err := projects.CreateProject(conn, uuid, email, body.Name, body.Description, body.Color)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, proj)
	}
}

func Update(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var body updateProjectBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		uuidVal, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		uuid, _ := uuidVal.(string)
		role, _ := roleVal.(string)
		isTeacher := role == "teacher" || role == "admin"

		proj, err := projects.UpdateProject(conn, id, uuid, isTeacher, body.Name, body.Description, body.Color)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, proj)
	}
}

func Delete(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		uuidVal, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		uuid, _ := uuidVal.(string)
		role, _ := roleVal.(string)
		isTeacher := role == "teacher" || role == "admin"

		err := projects.DeleteProject(conn, id, uuid, isTeacher)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "project deleted successfully"})
	}
}

func AddVM(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var body addVMBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		uuidVal, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		uuid, _ := uuidVal.(string)
		role, _ := roleVal.(string)
		isTeacher := role == "teacher" || role == "admin"

		if len(body.VMIDs) > 0 {
			err := projects.SetProjectVMs(conn, id, body.VMIDs, uuid, isTeacher)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "project VMs updated"})
			return
		}

		if body.VMID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vm_id"})
			return
		}

		err := projects.AddVMToProject(conn, id, body.VMID, uuid, isTeacher)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "VM added to project"})
	}
}

func RemoveVM(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		vmidStr := c.Param("vmid")
		vmid, err := strconv.Atoi(vmidStr)
		if err != nil || vmid <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vm_id"})
			return
		}

		uuidVal, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		uuid, _ := uuidVal.(string)
		role, _ := roleVal.(string)
		isTeacher := role == "teacher" || role == "admin"

		err = projects.RemoveVMFromProject(conn, id, vmid, uuid, isTeacher)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "VM removed from project"})
	}
}

func AssignVM(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		vmidStr := c.Param("id")
		vmid, err := strconv.Atoi(vmidStr)
		if err != nil || vmid <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vm id"})
			return
		}

		var body assignVMBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		uuidVal, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		uuid, _ := uuidVal.(string)
		role, _ := roleVal.(string)
		isTeacher := role == "teacher" || role == "admin"

		err = projects.AssignVMProject(conn, vmid, body.ProjectID, uuid, isTeacher)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "VM project assigned successfully"})
	}
}

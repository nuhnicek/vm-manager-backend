package teacher

import (
	"net/http"
	"vm-manager-backend/internal/core/teacher"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type createStudentBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}

type bulkActionBody struct {
	Action string `json:"action"`
}

func ListStudents(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		students, err := teacher.ListStudents(conn)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, students)
	}
}

func CreateStudent(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createStudentBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student payload: " + err.Error()})
			return
		}

		student, err := teacher.CreateStudent(conn, body.Email, body.Password, body.Role)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, student)
	}
}

func ListAllVMs(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		allVMs, err := vms.List(conn, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, allVMs)
	}
}

func BulkAction(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body bulkActionBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bulk action payload: " + err.Error()})
			return
		}

		err := teacher.TeacherBulkAction(conn, body.Action)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "completed", "action": body.Action})
	}
}

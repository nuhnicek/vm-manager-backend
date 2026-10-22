package auth

import (
	"fmt"
	"net/http"
	"vm-manager-backend/internal/core/auth"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body LoginRequest
		err := c.ShouldBindBodyWithJSON(&body)
		if err != nil {
			fmt.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		identifier := body.Email
		if identifier == "" {
			identifier = body.Username
		}
		if identifier == "" || body.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
			return
		}
		err, session, userDoc := auth.Login(identifier, body.Password, conn)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		role := userDoc.Role
		if role == "" {
			if userDoc.IsTeacher {
				role = "teacher"
			} else {
				role = "student"
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Login successful",
			"token":   session,
			"user":    identifier,
			"role":    role,
			"uuid":    userDoc.UUID,
		})
	}
}

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
	Password string `json:"password"`
}

func Login(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body LoginRequest
		err := c.ShouldBindBodyWithJSON(&body)
		if err != nil {

			fmt.Println(err)
			c.Status(http.StatusBadRequest)
			return
		}
		err, session := auth.Login(body.Email, body.Password, conn)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(200, gin.H{
			"token": session,
		})
	}
}

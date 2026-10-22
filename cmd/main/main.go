package main

import (
	"log"
	"vm-manager-backend/cmd/main/bootstrap"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	err, config := bootstrap.Initialize(router)

	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	if err := router.Run(config.Port); err != nil {
		log.Fatalf("Server exited with error: %v", err)
	}
}

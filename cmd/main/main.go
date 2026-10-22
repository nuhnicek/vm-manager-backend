package main

import (
	"vm-manager-backend/cmd/main/bootstrap"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	err, config := bootstrap.Initialize(router)

	if err != nil {
		panic(err)
	}

	router.Run(config.Port)
}

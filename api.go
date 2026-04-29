package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// StartAPIServer starts a web server to accept hashes
func StartAPIServer(manager *ManagerNode, port int) {
	r := gin.Default()

	// This tells Gin to look in the /static folder
	r.StaticFile("/", "./static/index.html")

	// Route to submit a hash
	r.POST("/crack", func(c *gin.Context) {
		var input struct {
			Hash string `json:"hash"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		// Tell the manager to start cracking this new hash
		manager.CurrentHash = input.Hash
		manager.FoundPassword = "" // Reset state

		c.JSON(http.StatusOK, gin.H{"status": "Cracking started", "hash": input.Hash})
	})

	// Route to check status
	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"hash":     manager.CurrentHash,
			"found":    manager.FoundPassword != "",
			"password": manager.FoundPassword,
		})
	})

	r.Run(fmt.Sprintf(":%d", port))
}

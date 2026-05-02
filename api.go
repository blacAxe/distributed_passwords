package main

import (
	"fmt"
	"net/http"
	"sync" // Added for safety

	"github.com/gin-gonic/gin"
)

var mu sync.Mutex

func StartAPIServer(manager *ManagerNode, port int) {
	r := gin.Default()

	r.StaticFile("/", "./static/index.html")

	r.POST("/crack", func(c *gin.Context) {
		var input struct {
			Hash string `json:"hash"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		mu.Lock()
		// Reset everything for a fresh run
		manager.CurrentHash = input.Hash
		manager.FoundPassword = ""
		manager.IsProcessing = false 
		
		// Kill any existing background worker calls
		if manager.CancelFunc != nil {
			manager.CancelFunc()
		}
		mu.Unlock()

		c.JSON(http.StatusOK, gin.H{"status": "Cracking started", "hash": input.Hash})
	})

	r.GET("/status", func(c *gin.Context) {
		mu.Lock()
		defer mu.Unlock()
		c.JSON(http.StatusOK, gin.H{
			"hash":     manager.CurrentHash,
			"found":    manager.FoundPassword != "",
			"password": manager.FoundPassword,
			"busy":     manager.IsProcessing,
		})
	})

	r.Run(fmt.Sprintf(":%d", port))
}
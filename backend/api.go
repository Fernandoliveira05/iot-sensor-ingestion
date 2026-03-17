package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Telemetry struct {
	DeviceID    string  `json:"device_id"`
	Timestamp   string  `json:"timestamp"`
	SensorType  string  `json:"sensor_type"`
	ReadingType string  `json:"reading_type"`
	Value       float64 `json:"value"`
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.POST("/telemetria", func(c *gin.Context) {
		var data Telemetry

		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "JSON não tá correto!",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Dados recebidos!",
			"data":    data,
		})
	})

	return r
}

func main() {
	r := setupRouter()
	r.Run(":8080")
}
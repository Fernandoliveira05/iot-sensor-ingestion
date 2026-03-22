package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

var db *sql.DB
var rabbitConn *amqp.Connection
var rabbitCh *amqp.Channel

type Telemetry struct {
	DeviceID    string      `json:"device_id"`
	SensorType  string      `json:"sensor_type"`
	ReadingType string      `json:"reading_type"`
	Value       interface{} `json:"value"`
	Timestamp   time.Time   `json:"timestamp"`
}

type TelemetryResponse struct {
	DeviceID      string         `json:"device_id"`
	Timestamp     time.Time      `json:"timestamp"`
	SensorType    string         `json:"sensor_type"`
	ReadingType   string         `json:"reading_type"`
	AnalogValue   sql.NullFloat64 `json:"-"`
	DiscreteValue sql.NullString  `json:"-"`
	Value         interface{}    `json:"value"`
}

var publishMessage func(Telemetry) error = sendToRabbitMQ

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func initRabbit() error {
	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		return fmt.Errorf("RABBIT_URL não definida")
	}

	queueName := getEnv("QUEUE_NAME", "task_queue")

	var err error

	for i := 1; i <= 10; i++ {
		rabbitConn, err = amqp.Dial(rabbitURL)
		if err == nil {
			rabbitCh, err = rabbitConn.Channel()
		}
		if err == nil {
			_, err = rabbitCh.QueueDeclare(queueName, true, false, false, false, nil)
		}
		if err == nil {
			return nil
		}

		log.Printf("Tentativa %d/10 para conectar ao RabbitMQ falhou: %v", i, err)
		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("erro ao conectar no RabbitMQ após várias tentativas: %w", err)
}

func initDB() error {
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		return fmt.Errorf("DB_URL não definida")
	}

	var err error

	for i := 1; i <= 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			return nil
		}

		log.Printf("Tentativa %d/10 para conectar ao banco falhou: %v", i, err)
		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("erro ao conectar ao banco após várias tentativas: %w", err)
}

func sendToRabbitMQ(message Telemetry) error {
	queueName := getEnv("QUEUE_NAME", "task_queue")

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("erro ao serializar mensagem: %w", err)
	}

	return rabbitCh.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.POST("/telemetria", func(c *gin.Context) {
		var t Telemetry

		if err := c.ShouldBindJSON(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
			return
		}

		if t.DeviceID == "" || t.SensorType == "" || t.ReadingType == "" || t.Timestamp.IsZero() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "campos obrigatórios ausentes"})
			return
		}

		if t.ReadingType != "analog" && t.ReadingType != "discrete" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "reading_type deve ser 'analog' ou 'discrete'"})
			return
		}

		err := publishMessage(t)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "erro ao enviar para fila: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"message": "Enviado para processamento assíncrono",
			"data":    t,
		})
	})

	r.GET("/telemetria", func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT device_id, timestamp, sensor_type, reading_type, analog_value, discrete_value
			FROM telemetria
			ORDER BY id DESC
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var registros []TelemetryResponse

		for rows.Next() {
			var t TelemetryResponse

			if err := rows.Scan(
				&t.DeviceID,
				&t.Timestamp,
				&t.SensorType,
				&t.ReadingType,
				&t.AnalogValue,
				&t.DiscreteValue,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if t.ReadingType == "analog" && t.AnalogValue.Valid {
				t.Value = t.AnalogValue.Float64
			} else if t.ReadingType == "discrete" && t.DiscreteValue.Valid {
				t.Value = t.DiscreteValue.String
			}

			registros = append(registros, t)
		}

		c.JSON(http.StatusOK, registros)
	})

	return r
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("Erro ao inicializar o banco de dados: %v", err)
	}
	defer db.Close()

	if err := initRabbit(); err != nil {
		log.Fatalf("Erro ao inicializar o RabbitMQ: %v", err)
	}
	defer rabbitCh.Close()
	defer rabbitConn.Close()

	r := setupRouter()
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
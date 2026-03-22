package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	DeviceID    string      `json:"device_id"`
	Timestamp   string      `json:"timestamp"`
	SensorType  string      `json:"sensor_type"`
	ReadingType string      `json:"reading_type"`
	Value       interface{} `json:"value"`
}

func initDB() (*sql.DB, error) {
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@postgres:5432/telemetria?sslmode=disable"
	}

	var db *sql.DB
	var err error

	for i := 1; i <= 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			log.Println("Conectado ao PostgreSQL")
			return db, nil
		}

		log.Printf("Tentativa %d/10 para conectar ao PostgreSQL falhou: %v", i, err)
		time.Sleep(3 * time.Second)
	}

	return nil, fmt.Errorf("erro ao conectar ao PostgreSQL após várias tentativas: %w", err)
}

func connectRabbitMQ() (*amqp.Connection, *amqp.Channel, string, error) {
	url := os.Getenv("RABBIT_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
	}

	queueName := os.Getenv("QUEUE_NAME")
	if queueName == "" {
		queueName = "task_queue"
	}

	var conn *amqp.Connection
	var ch *amqp.Channel
	var err error

	for i := 1; i <= 15; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			ch, err = conn.Channel()
		}
		if err == nil {
			_, err = ch.QueueDeclare(
				queueName,
				true,
				false,
				false,
				false,
				nil,
			)
		}
		if err == nil {
			log.Println("Conectado ao RabbitMQ")
			return conn, ch, queueName, nil
		}

		log.Printf("Tentativa %d/15 para conectar ao RabbitMQ falhou: %v", i, err)
		time.Sleep(2 * time.Second)
	}

	return nil, nil, "", fmt.Errorf("erro ao conectar ao RabbitMQ após várias tentativas: %w", err)
}

func consumeMessages(db *sql.DB) error {
	conn, ch, queueName, err := connectRabbitMQ()
	if err != nil {
		return err
	}
	defer conn.Close()
	defer ch.Close()

	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("erro ao registrar consumer: %w", err)
	}

	log.Println("Aguardando mensagens...")

	for msg := range msgs {
		log.Printf("Mensagem recebida: %s", msg.Body)

		var payload Message
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			log.Printf("erro ao decodificar JSON: %v", err)
			msg.Nack(false, false)
			continue
		}

		if err := saveTelemetry(db, payload); err != nil {
			log.Printf("erro ao inserir no banco: %v", err)
			msg.Nack(false, true)
			continue
		}

		msg.Ack(false)
		log.Println("Mensagem salva no PostgreSQL")
	}

	return nil
}

func saveTelemetry(db *sql.DB, payload Message) error {
	parsedTime, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		return err
	}

	var analogValue sql.NullFloat64
	var discreteValue sql.NullString

	switch v := payload.Value.(type) {
	case float64:
		analogValue = sql.NullFloat64{
			Float64: v,
			Valid:   true,
		}
	case string:
		discreteValue = sql.NullString{
			String: v,
			Valid:  true,
		}
	case bool:
		if v {
			discreteValue = sql.NullString{String: "true", Valid: true}
		} else {
			discreteValue = sql.NullString{String: "false", Valid: true}
		}
	default:
		discreteValue = sql.NullString{Valid: false}
	}

	_, err = db.Exec(`
		INSERT INTO telemetria (
			device_id,
			timestamp,
			sensor_type,
			reading_type,
			analog_value,
			discrete_value
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		payload.DeviceID,
		parsedTime,
		payload.SensorType,
		payload.ReadingType,
		analogValue,
		discreteValue,
	)

	return err
}

func main() {
	db, err := initDB()
	if err != nil {
		log.Fatalf("erro ao inicializar banco: %v", err)
	}
	defer db.Close()

	for {
		err := consumeMessages(db)
		log.Printf("Erro no consumer, reiniciando: %v", err)
		time.Sleep(5 * time.Second)
	}
}
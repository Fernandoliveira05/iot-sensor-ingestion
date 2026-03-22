package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type TelemetryMessage struct {
	DeviceID    string      `json:"device_id"`
	SensorType  string      `json:"sensor_type"`
	ReadingType string      `json:"reading_type"`
	Value       interface{} `json:"value"`
	Timestamp   time.Time   `json:"timestamp"`
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func main() {
	amqpURL := getEnv("RABBIT_URL", "amqp://guest:guest@rabbitmq:5672/")
	queueName := getEnv("QUEUE_NAME", "task_queue")

	totalMessages := 50000
	concurrency := 20

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("erro ao conectar no RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("erro ao abrir channel: %v", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("erro ao declarar fila: %v", err)
	}

	jobs := make(chan int, 1000)
	var sent uint64
	var failed uint64

	start := time.Now()
	var wg sync.WaitGroup

	for w := 0; w < concurrency; w++ {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			workerCh, err := conn.Channel()
			if err != nil {
				log.Printf("worker %d: erro ao abrir channel: %v", workerID, err)
				atomic.AddUint64(&failed, 1)
				return
			}
			defer workerCh.Close()

			for i := range jobs {
				var value interface{}
				readingType := "analog"
				sensorType := "temperature"

				if i%5 == 0 {
					readingType = "discrete"
					sensorType = "presence"
					if i%2 == 0 {
						value = "present"
					} else {
						value = "absent"
					}
				} else {
					value = 20 + float64(i%15)
				}

				msg := TelemetryMessage{
					DeviceID:    fmt.Sprintf("device-%03d", i%100),
					SensorType:  sensorType,
					ReadingType: readingType,
					Value:       value,
					Timestamp:   time.Now().UTC(),
				}

				body, err := json.Marshal(msg)
				if err != nil {
					log.Printf("worker %d: erro ao serializar JSON: %v", workerID, err)
					atomic.AddUint64(&failed, 1)
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err = workerCh.PublishWithContext(
					ctx,
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
				cancel()

				if err != nil {
					log.Printf("worker %d: erro ao publicar msg %d: %v", workerID, i, err)
					atomic.AddUint64(&failed, 1)
					continue
				}

				newSent := atomic.AddUint64(&sent, 1)
				if newSent%5000 == 0 {
					log.Printf("%d mensagens enviadas...", newSent)
				}
			}
		}(w)
	}

	for i := 1; i <= totalMessages; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()

	elapsed := time.Since(start)
	sentCount := atomic.LoadUint64(&sent)
	failedCount := atomic.LoadUint64(&failed)

	fmt.Println("===== RESULTADO =====")
	fmt.Printf("Mensagens enviadas com sucesso: %d\n", sentCount)
	fmt.Printf("Mensagens com falha: %d\n", failedCount)
	fmt.Printf("Tempo total: %s\n", elapsed)

	if elapsed.Seconds() > 0 {
		fmt.Printf("Throughput: %.2f msg/s\n", float64(sentCount)/elapsed.Seconds())
	}
}
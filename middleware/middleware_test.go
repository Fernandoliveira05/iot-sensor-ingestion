package main

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSaveTelemetry_AnalogValue(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("erro ao criar sqlmock: %v", err)
	}
	defer db.Close()

	payload := Message{
		DeviceID:    "device-1",
		Timestamp:   "2026-03-23T10:00:00Z",
		SensorType:  "temperature",
		ReadingType: "analog",
		Value:       25.7,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO telemetria (
			device_id,
			timestamp,
			sensor_type,
			reading_type,
			analog_value,
			discrete_value
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)).
		WithArgs(
			payload.DeviceID,
			Parse(t, payload.Timestamp),
			payload.SensorType,
			payload.ReadingType,
			sql.NullFloat64{Float64: 25.7, Valid: true},
			sql.NullString{Valid: false},
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = saveTelemetry(db, payload)
	if err != nil {
		t.Fatalf("esperava nada, recebeu erro: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas ñ atendidas: %v", err)
	}
}

func TestSaveTelemetry_DiscreteStringValue(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("erro ao criar sqlmock: %v", err)
	}
	defer db.Close()

	payload := Message{
		DeviceID:    "device-2",
		Timestamp:   "2026-03-23T10:05:00Z",
		SensorType:  "presence",
		ReadingType: "discrete",
		Value:       "detected",
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO telemetria (
			device_id,
			timestamp,
			sensor_type,
			reading_type,
			analog_value,
			discrete_value
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)).
		WithArgs(
			payload.DeviceID,
			Parse(t, payload.Timestamp),
			payload.SensorType,
			payload.ReadingType,
			sql.NullFloat64{Valid: false},
			sql.NullString{String: "detected", Valid: true},
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = saveTelemetry(db, payload)
	if err != nil {
		t.Fatalf("esperava nada, recebeu erro: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas ñ atendidas: %v", err)
	}
}

func TestSaveTelemetry_DiscreteBoolTrue(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("erro ao criar sqlmock: %v", err)
	}
	defer db.Close()

	payload := Message{
		DeviceID:    "device-3",
		Timestamp:   "2026-03-23T10:10:00Z",
		SensorType:  "switch",
		ReadingType: "discrete",
		Value:       true,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO telemetria (
			device_id,
			timestamp,
			sensor_type,
			reading_type,
			analog_value,
			discrete_value
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)).
		WithArgs(
			payload.DeviceID,
			Parse(t, payload.Timestamp),
			payload.SensorType,
			payload.ReadingType,
			sql.NullFloat64{Valid: false},
			sql.NullString{String: "true", Valid: true},
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = saveTelemetry(db, payload)
	if err != nil {
		t.Fatalf("esperava nada, recebeu erro: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("ñ atendidas: %v", err)
	}
}

func TestSaveTelemetry_InvalidTimestamp(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("erro ao criar sqlmock: %v", err)
	}
	defer db.Close()

	payload := Message{
		DeviceID:    "device-4",
		Timestamp:   "data-invalida",
		SensorType:  "temperature",
		ReadingType: "analog",
		Value:       20.0,
	}

	err = saveTelemetry(db, payload)
	if err == nil {
		t.Fatal("esperava erro de timestamp inválido, mas recebeu nil")
	}
}

func Parse(t *testing.T, s string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("erro ao converter timestamp no teste: %v", err)
	}

	return parsed
}
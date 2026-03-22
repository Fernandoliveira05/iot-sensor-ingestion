CREATE TABLE IF NOT EXISTS telemetria (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    sensor_type VARCHAR(50) NOT NULL,
    reading_type VARCHAR(20) NOT NULL,
    analog_value DOUBLE PRECISION,
    discrete_value VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (reading_type IN ('analog', 'discrete')),
    CHECK (
        (reading_type = 'analog' AND analog_value IS NOT NULL)
        OR
        (reading_type = 'discrete' AND discrete_value IS NOT NULL)
    )
);
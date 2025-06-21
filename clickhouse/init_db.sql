CREATE DATABASE IF NOT EXISTS events;

CREATE TABLE IF NOT EXISTS events.tts (
    timestamp DateTime64(3) DEFAULT now64(3),
    service_id String,
    language LowCardinality(String),
    speaker LowCardinality(String),
) ENGINE = MergeTree()
ORDER BY (timestamp, service_id)
PARTITION BY toYYYYMM(timestamp);
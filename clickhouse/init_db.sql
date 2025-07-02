CREATE DATABASE IF NOT EXISTS events;

CREATE TABLE IF NOT EXISTS events.tts (
    timestamp DateTime64(3) DEFAULT now64(3),
    service_id String,
    language LowCardinality(String),
    speaker LowCardinality(String),
) ENGINE = MergeTree()
ORDER BY (timestamp, service_id)
PARTITION BY toYYYYMM(timestamp);

CREATE TABLE IF NOT EXISTS events.commands (
    timestamp DateTime64(3) DEFAULT now64(3),
    user_id String,
    type LowCardinality(String),
    name LowCardinality(String),
) ENGINE = MergeTree()
ORDER BY (timestamp, name)
PARTITION BY toYYYYMM(timestamp);

CREATE TABLE IF NOT EXISTS events.actions (
    timestamp DateTime64(3) DEFAULT now64(3),
    guild_id String,
    type LowCardinality(String),
    name LowCardinality(String),
) ENGINE = MergeTree()
ORDER BY (timestamp, name)
PARTITION BY toYYYYMM(timestamp);
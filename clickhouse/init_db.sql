CREATE DATABASE IF NOT EXISTS events;

CREATE TABLE IF NOT EXISTS events.tts (
    timestamp DateTime64(3) DEFAULT now64(3),
    service_id LowCardinality(String),
    language LowCardinality(String),
    speaker LowCardinality(String),
    chars UInt16,
    transed Boolean DEFAULT false
) ENGINE = MergeTree()
ORDER BY (service_id, timestamp)
PARTITION BY toYYYYMM(timestamp);

CREATE TABLE IF NOT EXISTS events.commands (
    timestamp DateTime64(3) DEFAULT now64(3),
    service_id LowCardinality(String),
    user_id String,
    type LowCardinality(String),
    name LowCardinality(String)
) ENGINE = MergeTree()
ORDER BY (service_id, name, timestamp)
PARTITION BY toYYYYMM(timestamp);

CREATE TABLE IF NOT EXISTS events.actions (
    timestamp DateTime64(3) DEFAULT now64(3),
    service_id LowCardinality(String),
    guild_id String,
    name LowCardinality(String)
) ENGINE = MergeTree()
ORDER BY (service_id, name, timestamp)
PARTITION BY toYYYYMM(timestamp);
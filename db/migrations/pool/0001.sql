-- +migrate Down
DROP SCHEMA IF EXISTS pool CASCADE;

-- +migrate Up
CREATE SCHEMA pool;

CREATE TABLE pool.transaction
(
    id              SERIAL PRIMARY KEY,
    hash            VARCHAR NOT NULL,
    received_at     TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    from_address    VARCHAR NOT NULL,
    gas_price       DECIMAL(78, 0),
    nonce           DECIMAL(78, 0),
    status          VARCHAR,
    ip              VARCHAR,
    encoded         VARCHAR,
    decoded         jsonb,
    error           VARCHAR
);

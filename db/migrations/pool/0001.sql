-- +migrate Down
DROP SCHEMA IF EXISTS pool CASCADE;

-- +migrate Up
CREATE SCHEMA pool;

CREATE TABLE pool.transaction
(
    hash                   VARCHAR PRIMARY KEY,
    received_at            TIMESTAMP WITH TIME ZONE NOT NULL,
    from_address           VARCHAR NOT NULL,
    gas_price              DECIMAL(78, 0),
    nonce                  DECIMAL(78, 0),
    status                 VARCHAR,
    encoded                VARCHAR,
    decoded                jsonb
);

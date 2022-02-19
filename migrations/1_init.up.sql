DO $$ BEGIN
  CREATE EXTENSION pgcrypto;
  CREATE TYPE role AS ENUM ('ADMIN', 'CUSTOMER');
EXCEPTION
  WHEN duplicate_object THEN null;
END $$;

DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;

CREATE TABLE products (
   id UUID NOT NULL default gen_random_uuid(),
   name TEXT NOT NULL,
   description TEXT NULL,
   user_id UUID NOT NULL,
   created_at TIMESTAMP NOT NULL,
   updated_at TIMESTAMP NOT NULL,
   deleted BOOLEAN NOT NULL default FALSE,
   primary key(id)
);

CREATE TABLE users (
   id UUID NOT NULL default gen_random_uuid(),
   email TEXT NOT NULL UNIQUE,
   name VARCHAR(100) NOT NULL,
   password TEXT NOT NULL,
   role role NOT NULL,
   created_at TIMESTAMP NOT NULL,
   updated_at TIMESTAMP NOT NULL,
   deleted BOOLEAN NOT NULL default FALSE,
   primary key(id)
);

INSERT INTO users (email, password, name, role, created_at, updated_at) VALUES ('frtzsm@gmail.com', crypt('177013', gen_salt('bf', 8)), 'Administrator', 'ADMIN', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO users (email, password, name, role, created_at, updated_at) VALUES ('fahrulrozi1288@gmail.com', crypt('322420', gen_salt('bf', 8)), 'Fahrul', 'CUSTOMER', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

CREATE INDEX products_user_idx ON products(user_id);
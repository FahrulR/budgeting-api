DROP TABLE IF EXISTS incomes;

CREATE TABLE incomes (
   id UUID NOT NULL default gen_random_uuid(),
   name TEXT NOT NULL,
   description TEXT NULL,
   "date" DATE NOT NULL,
   currency VARCHAR(3) NOT NULL,
   amount DECIMAL(12,2) NOT NULL,
   user_id UUID NOT NULL,
   created_at TIMESTAMP NOT NULL,
   updated_at TIMESTAMP NOT NULL,
   deleted BOOLEAN NOT NULL default FALSE,
   primary key(id)
);

CREATE INDEX incomes_user_idx ON incomes(user_id);
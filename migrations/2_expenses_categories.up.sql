DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS categories;

CREATE TABLE expenses (
   id UUID NOT NULL default gen_random_uuid(),
   product_id UUID NOT NULL,
   "date" DATE NOT NULL,
   cash MONEY NOT NULL,
   user_id UUID NOT NULL,
   created_at TIMESTAMP NOT NULL,
   updated_at TIMESTAMP NOT NULL,
   deleted BOOLEAN NOT NULL default FALSE,
   primary key(id)
);

CREATE TABLE categories (
   id UUID NOT NULL default gen_random_uuid(),
   name TEXT NOT NULL,
   description TEXT NULL,
   user_id UUID NOT NULL,
   created_at TIMESTAMP NOT NULL,
   updated_at TIMESTAMP NOT NULL,
   deleted BOOLEAN NOT NULL default FALSE,
   primary key(id)
);

ALTER TABLE products ADD category_id UUID NOT NULL;
CREATE INDEX products_category_idx ON products(category_id);
CREATE INDEX expenses_product_idx ON expenses(product_id);
CREATE INDEX expenses_user_idx ON expenses(user_id);
CREATE INDEX categories_user_idx ON categories(user_id);

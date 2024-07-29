CREATE TABLE IF NOT EXISTS recipes (
	recipe_id integer PRIMARY KEY,
	crafted_item_id integer REFERENCES items (item_id) ON DELETE CASCADE,
	crafted_item_count integer NOT NULL
);

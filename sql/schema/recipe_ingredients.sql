CREATE TABLE IF NOT EXISTS recipe_ingredients (
	recipe_id integer REFERENCES recipes ON DELETE CASCADE, 
	ingredient_id integer REFERENCES items (item_id) ON DELETE CASCADE,
	quantity integer NOT NULL,
	PRIMARY KEY (recipe_id, ingredient_id)
);

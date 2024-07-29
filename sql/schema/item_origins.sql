CREATE TABLE IF NOT EXISTS item_origins (
		item_id integer REFERENCES items ON DELETE CASCADE,
		origin text NOT NULL,
		PRIMARY KEY (item_id, origin)
);

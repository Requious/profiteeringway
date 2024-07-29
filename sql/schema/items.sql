CREATE TABLE IF NOT EXISTS items (
		item_id integer PRIMARY KEY,
		type text NOT NULL,
		name text NOT NULL,
		item_level integer,
		special_currency_item_id integer,
		special_currency_count integer,
		high_qualityable boolean NOT NULL,
		marketable boolean NOT NULL,
		gil_price integer,
		class_job_restriction text
);

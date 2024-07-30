CREATE TABLE IF NOT EXISTS prices (
	price_id bigserial PRIMARY KEY,
	item_id integer REFERENCES items ON DELETE CASCADE NOT NULL,
	world_id integer REFERENCES worlds ON DELETE RESTRICT NOT NULL,
	update_time timestamp without time zone NOT NULL,
	nq_sale_velocity integer NOT NULL,
	hq_sale_velocity integer NOT NULL,
	min_price_nq integer NOT NULL,
	min_price_hq integer NOT NULL
); --TODO: configure partitions by update_time

CREATE TABLE IF NOT EXISTS listings (
	listing_id bigserial PRIMARY KEY,
	price_id integer REFERENCES prices ON DELETE CASCADE NOT NULL,
	price_per_unit integer NOT NULL,
	quantity integer NOT NULL,
	high_quality boolean NOT NULL
);

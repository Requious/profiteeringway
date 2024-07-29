CREATE TABLE IF NOT EXISTS worlds (
	world_id integer PRIMARY KEY,
	name text NOT NULL,
	datacenter text NOT NULL,
	is_public boolean NOT NULL
);

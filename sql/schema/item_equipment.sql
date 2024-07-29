CREATE TABLE IF NOT EXISTS item_equipment_types (
	item_id integer REFERENCES items ON DELETE CASCADE,
	equipment_type text NOT NULL,
	PRIMARY KEY (item_id, equipment_type)
);

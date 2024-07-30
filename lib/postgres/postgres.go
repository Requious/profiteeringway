package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

const initializePriceTables = `CREATE TABLE IF NOT EXISTS prices (
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
);`

type Postgres struct {
	Db *sql.DB
}

func NewPostgres(connStr string) (*Postgres, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Postgres connection: %s", err)
	}
	return &Postgres{
		Db: db,
	}, nil
}

type ItemForRead struct {
	ItemID                *sql.NullInt32
	Type                  *sql.NullString
	Name                  *sql.NullString
	ItemLevel             *sql.NullInt32
	SpecialCurrencyItemID *sql.NullInt32
	SpecialCurrencyCount  *sql.NullInt32
	HighQualityable       *sql.NullBool
	Marketable            *sql.NullBool
	GilPrice              *sql.NullInt32
	ClassJobRestriction   *sql.NullString
}

func (p *Postgres) SelectItemWithID(id int32) (*ItemForRead, error) {
	i := ItemForRead{
		ItemID:                &sql.NullInt32{},
		Type:                  &sql.NullString{},
		Name:                  &sql.NullString{},
		ItemLevel:             &sql.NullInt32{},
		SpecialCurrencyItemID: &sql.NullInt32{},
		SpecialCurrencyCount:  &sql.NullInt32{},
		HighQualityable:       &sql.NullBool{},
		Marketable:            &sql.NullBool{},
		GilPrice:              &sql.NullInt32{},
		ClassJobRestriction:   &sql.NullString{},
	}

	row := p.Db.QueryRow("SELECT * FROM items WHERE item_id = $1", id)

	if err := row.Scan(i.ItemID, i.Type, i.Name, i.ItemLevel, i.SpecialCurrencyItemID, i.SpecialCurrencyCount, i.HighQualityable, i.Marketable, i.GilPrice, i.ClassJobRestriction); err != nil {
		return nil, fmt.Errorf("failed to retrieve row for item %v: %w", id, err)
	}
	return &i, nil
}

func (p *Postgres) InitializePriceTables() error {
	_, err := p.Db.Exec(initializePriceTables)
	if err != nil {
		return fmt.Errorf("failed to initialize price tables: %w", err)
	}
	return nil
}

func (p *Postgres) CleanUp() error {
	return p.Db.Close()
}

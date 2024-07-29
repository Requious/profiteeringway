package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

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

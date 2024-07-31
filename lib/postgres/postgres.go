package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"profiteeringway/lib/universalis"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
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
	Db     *sql.DB
	logger *zap.SugaredLogger
}

func NewPostgres(connStr string, logger *zap.SugaredLogger) (*Postgres, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Postgres connection: %s", err)
	}
	return &Postgres{
		Db:     db,
		logger: logger,
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

/*
prices

	price_id bigserial PRIMARY KEY,
	item_id integer REFERENCES items ON DELETE CASCADE NOT NULL,
	world_id integer REFERENCES worlds ON DELETE RESTRICT NOT NULL,
	update_time timestamp without time zone NOT NULL,
	nq_sale_velocity integer NOT NULL,
	hq_sale_velocity integer NOT NULL,
	min_price_nq integer NOT NULL,
	min_price_hq integer NOT NULL

listings

	listing_id bigserial PRIMARY KEY,
	price_id integer REFERENCES prices ON DELETE CASCADE NOT NULL,
	price_per_unit integer NOT NULL,
	quantity integer NOT NULL,
	high_quality boolean NOT NULL
*/

func checkPositive(nums []int) bool {
	for _, num := range nums {
		if num <= 0 {
			return false
		}
	}
	return true
}
func (p *Postgres) WriteUniversalisPriceData(ctx context.Context, upd *universalis.UniversalisPriceData) error {
	successCount := 0
	for _, priceData := range upd.Items {
		// Check in case it's garbage
		positive := checkPositive([]int{priceData.ItemID, priceData.WorldID})
		if !positive {
			p.logger.Warnf("unexpected negative values found: %+v", priceData)
			continue
		}
		rows, err := p.Db.QueryContext(ctx, `INSERT INTO prices
(item_id, world_id, update_time, nq_sale_velocity, hq_sale_velocity, min_price_nq, min_price_hq)
VALUES ($1, $2, to_timestamp($3::double precision/1000), $4, $5, $6, $7) 
RETURNING price_id;`, priceData.ItemID, priceData.WorldID, priceData.LastUploadTime, priceData.NqSaleVelocity, priceData.HqSaleVelocity, priceData.MinPriceNQ, priceData.MinPriceHQ)

		if err != nil {
			p.logger.Errorf("failed to write price: %s", err)
			continue
		}

		successCount += 1

		var priceID int
		for rows.Next() {
			if err := rows.Scan(&priceID); err != nil {
				p.logger.Errorf("failed to scan price_id out of insert command: %s", err)
				continue
			}
		}

		for _, l := range priceData.Listings {
			_, err := p.Db.ExecContext(ctx, `INSERT INTO listings (price_id, price_per_unit, quantity, high_quality) VALUES ($1, $2, $3, $4)`,
				priceID, l.PricePerUnit, l.Quantity, l.Hq)
			if err != nil {
				p.logger.Errorf("failed to write listing: %s", err)
				continue
			}
			successCount += 1
		}
	}

	if successCount == 0 {
		return fmt.Errorf("all writes failed, see logs")
	}
	return nil
}

func (p *Postgres) NorthAmericanWorlds() ([]int, error) {
	rows, err := p.Db.Query(`SELECT world_id FROM worlds WHERE datacenter IN ('Aether', 'Primal', 'Crystal', 'Dynamis') AND is_public;`)
	if err != nil {
		return nil, fmt.Errorf("failed to get NA worlds: %w", err)
	}

	var worldIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data for world id: %w", err)
		}
		worldIDs = append(worldIDs, id)
	}
	return worldIDs, nil
}

func (p *Postgres) GetItemIDsForStaticQuery(query string) ([]int, error) {
	rows, err := p.Db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get NA worlds: %w", err)
	}

	var itemIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data for world id: %w", err)
		}
		itemIDs = append(itemIDs, id)
	}
	return itemIDs, nil
}

func (p *Postgres) DawntrailMateriaIDs() ([]int, error) {
	return p.GetItemIDsForStaticQuery(`SELECT item_id FROM items WHERE type IN ('Materia') AND item_level > 650;`)
}

func (p *Postgres) DawntrailConsumables() ([]int, error) {
	return p.GetItemIDsForStaticQuery(`SELECT item_id FROM items WHERE type IN ('Meal', 'Medicine') AND item_level > 700;`)
}

func (p *Postgres) DawntrailTierOneCraftedEquipment() ([]int, error) {
	return p.GetItemIDsForStaticQuery(`SELECT item_id FROM items WHERE type IN ('Marauder''s Arm','Two–handed Thaumaturge''s Arm','Weaver''s Primary Tool','Goldsmith''s Secondary Tool','Botanist''s Secondary Tool','Astrologian''s Arm','Fisher''s Primary Tool','Alchemist''s Primary Tool','Archer''s Arm','One–handed Conjurer''s Arm','Blacksmith''s Primary Tool','Arcanist''s Grimoire','Goldsmith''s Primary Tool','Alchemist''s Secondary Tool','Gladiator''s Arm','Red Mage''s Arm','Leatherworker''s Primary Tool','Scholar''s Arm','Earrings','Sage''s Arm','Blue Mage''s Arm','Rogue''s Arm','Blacksmith''s Secondary Tool','Culinarian''s Primary Tool','Reaper''s Arm','Miner''s Secondary Tool','Botanist''s Primary Tool','Culinarian''s Secondary Tool','Weaver''s Secondary Tool','Dancer''s Arm','Carpenter''s Secondary Tool','Armorer''s Primary Tool','Carpenter''s Primary Tool','Two–handed Conjurer''s Arm','Armorer''s Secondary Tool','One–handed Thaumaturge''s Arm','Dark Knight''s Arm','Miner''s Primary Tool','Samurai''s Arm','Shield','Fisher''s Secondary Tool','Machinist''s Arm','Hands','Body','Head','Necklace','Ring','Legs','Feet','Bracelets','Leatherworker''s Secondary Tool','Pugilist''s Arm','Lancer''s Arm','Pictomancer''s Arm','Viper''s Arm','Gunbreaker''s Arm') AND item_level > 709 AND marketable`)
}

func (p *Postgres) DawntrailMaterialsSetOne() ([]int, error) {
	return p.GetItemIDsForStaticQuery(`SELECT item_id FROM items WHERE type IN ('Reagent','Ingredient','Seafood','Crystal','Metal','Stone','Lumber','Bone') AND item_level > 679 AND marketable;`)
}

// We split these up to stay below the 100 item threshold for Universalis data.
func (p *Postgres) DawntrailMaterialsSetTwo() ([]int, error) {
	return p.GetItemIDsForStaticQuery(`SELECT item_id FROM items WHERE type IN ('Leather','Cloth') AND item_level > 679;`)
}

type PriceRow struct {
	Name       string
	WorldName  string
	MinPriceHQ int
	MinPriceNQ int
}

func (p *Postgres) GetItemPricesFromItemID(ctx context.Context, itemID int) ([]*PriceRow, error) {
	rows, err := p.Db.QueryContext(ctx, `SELECT
	name,
	world_name,
	MIN(overall_min_price_hq) AS min_price_hq,
	MIN(overall_min_price_nq) AS min_price_nq
FROM
(SELECT
	items.name,
	price_world.world_name,
	MIN(price_world.min_price_hq) OVER (PARTITION BY price_world.world_name) AS overall_min_price_hq,
	MIN(price_world.min_price_nq) OVER (PARTITION BY price_world.world_name) AS overall_min_price_nq,
	rank() OVER (PARTITION BY price_world.update_time ORDER BY price_world.update_time DESC) AS recency_rank
FROM
	items RIGHT JOIN (
		SELECT
			prices.item_id,
			worlds.name AS world_name,
			prices.min_price_hq,
			prices.min_price_nq,
			prices.update_time
		FROM
			prices INNER JOIN worlds USING (world_id)
		WHERE
			prices.min_price_hq <> 0
	) price_world USING (item_id)
WHERE
	items.item_id = ($1)
)
WHERE
	recency_rank = 1
GROUP BY
	name, world_name
ORDER BY 
	min_price_hq;`, itemID)
	if err != nil {
		return nil, fmt.Errorf("Postgres error: %s", err)
	}

	var prices []*PriceRow
	for rows.Next() {
		var minPriceHQ, minPriceNQ int
		var name, worldName string
		if err := rows.Scan(&name, &worldName, &minPriceHQ, &minPriceNQ); err != nil {
			return nil, fmt.Errorf("Failed to scan values out of SQL row: %w", err)
		}
		prices = append(prices, &PriceRow{
			Name:       name,
			WorldName:  worldName,
			MinPriceHQ: minPriceHQ,
			MinPriceNQ: minPriceNQ,
		})
	}
	return prices, nil
}

// Reminder this is case insensitive lookup with UPPER(name) = UPPER(db.name)
func (p *Postgres) GetItemPricesFromItemName(ctx context.Context, itemName string) ([]*PriceRow, error) {
	rows, err := p.Db.QueryContext(ctx, `SELECT
	name,
	world_name,
	MIN(overall_min_price_hq) AS min_price_hq,
	MIN(overall_min_price_nq) AS min_price_nq
FROM
(SELECT
	items.name,
	price_world.world_name,
	MIN(price_world.min_price_hq) OVER (PARTITION BY price_world.world_name) AS overall_min_price_hq,
	MIN(price_world.min_price_nq) OVER (PARTITION BY price_world.world_name) AS overall_min_price_nq,
	rank() OVER (PARTITION BY price_world.update_time ORDER BY price_world.update_time DESC) AS recency_rank
FROM
	items RIGHT JOIN (
		SELECT
			prices.item_id,
			worlds.name AS world_name,
			prices.min_price_hq,
			prices.min_price_nq,
			prices.update_time
		FROM
			prices INNER JOIN worlds USING (world_id)
		WHERE
			prices.min_price_hq <> 0
	) price_world USING (item_id)
WHERE
	UPPER(items.name) = UPPER(($1))
)
WHERE
	recency_rank = 1
GROUP BY
	name, world_name
ORDER BY 
	min_price_hq;`, itemName)
	if err != nil {
		return nil, fmt.Errorf("Postgres error: %s", err)
	}

	var prices []*PriceRow
	for rows.Next() {
		var minPriceHQ, minPriceNQ int
		var name, worldName string
		if err := rows.Scan(&name, &worldName, &minPriceHQ, &minPriceNQ); err != nil {
			return nil, fmt.Errorf("Failed to scan values out of SQL row: %w", err)
		}
		prices = append(prices, &PriceRow{
			Name:       name,
			WorldName:  worldName,
			MinPriceHQ: minPriceHQ,
			MinPriceNQ: minPriceNQ,
		})
	}
	return prices, nil
}

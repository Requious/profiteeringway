package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func (p *Postgres) MateriaPrices(ctx context.Context) ([]*HQPriceRow, error) {
	return nil, nil
}

func (p *Postgres) DawntrailTierOneCraftedEquipmentPrices(ctx context.Context) ([]*HQPriceRow, error) {
	return nil, nil
}

func (p *Postgres) DawntrailMaterialPrices(ctx context.Context) ([]*HQPriceRow, error) {
	return nil, nil
}

func (p *Postgres) GetPricesForItemIDs(ctx context.Context, itemIDs []int) ([]*HQPriceRow, error) {
	var stringIDs []string
	for _, itemID := range itemIDs {
		stringIDs = append(stringIDs, strconv.Itoa(itemID))
	}
	query := recentAllWorldsPriceQuery(fmt.Sprintf(`item_id IN (%s)`, strings.Join(stringIDs, ",")))
	rows, err := p.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("Postgres error: %s", err)
	}

	var prices []*HQPriceRow
	for rows.Next() {
		var minPriceHQ, minPriceNQ int
		var name, worldName string
		if err := rows.Scan(&name, &worldName, &minPriceHQ, &minPriceNQ); err != nil {
			return nil, fmt.Errorf("Failed to scan values out of SQL row: %w", err)
		}
		prices = append(prices, &HQPriceRow{
			Name:       name,
			WorldName:  worldName,
			MinPriceHQ: minPriceHQ,
			MinPriceNQ: minPriceNQ,
		})
	}
	return prices, nil
}

func recentAllWorldsPriceQuery(whereClause string) string {
	return fmt.Sprintf(`SELECT
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
	%s
)
WHERE
	recency_rank = 1
GROUP BY
	name, world_name
ORDER BY 
	min_price_hq;`, whereClause)
}

type AllWorldsPriceRowExpensive struct {
	Name        string
	WorldName   string
	Datacenter  string
	MinPrice    int
	HighQuality bool
}

func (p *Postgres) GetPriceForItemIDExpensive(ctx context.Context, itemID int) ([]*AllWorldsPriceRowExpensive, error) {
	query := recentAllWorldsPriceQueryExpensive("items.item_id = ($1)")
	rows, err := p.Db.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get prices for item (expensive query): %w", err)
	}

	var prices []*AllWorldsPriceRowExpensive
	for rows.Next() {
		var name, worldName, datacenter string
		var minPrice int
		var highQuality bool
		if err := rows.Scan(&name, &worldName, &datacenter, &minPrice, &highQuality); err != nil {
			return nil, fmt.Errorf("failed to scan out values into row: %w", err)
		}
		prices = append(prices, &AllWorldsPriceRowExpensive{
			Name:        name,
			WorldName:   worldName,
			Datacenter:  datacenter,
			MinPrice:    minPrice,
			HighQuality: highQuality,
		})
	}

	return prices, nil
}

func (p *Postgres) GetPriceForItemNameExpensive(ctx context.Context, itemName string) ([]*AllWorldsPriceRowExpensive, error) {
	query := recentAllWorldsPriceQueryExpensive("UPPER(items.name) = UPPER(($1))")
	rows, err := p.Db.QueryContext(ctx, query, itemName)
	if err != nil {
		return nil, fmt.Errorf("failed to get prices for item (expensive query): %w", err)
	}

	var prices []*AllWorldsPriceRowExpensive
	for rows.Next() {
		var name, worldName, datacenter string
		var minPrice int
		var highQuality bool
		if err := rows.Scan(&name, &worldName, &datacenter, &minPrice, &highQuality); err != nil {
			return nil, fmt.Errorf("failed to scan out values into row: %w", err)
		}
		prices = append(prices, &AllWorldsPriceRowExpensive{
			Name:        name,
			WorldName:   worldName,
			Datacenter:  datacenter,
			MinPrice:    minPrice,
			HighQuality: highQuality,
		})
	}

	return prices, nil
}

// The distinction being that we look through the listings ourselves to compute minimum prices.
func recentAllWorldsPriceQueryExpensive(whereClause string) string {
	return fmt.Sprintf(`SELECT
	name,
	world_name,
	datacenter,
	MIN(world_minimum_price) as min_price,
	high_quality
FROM (SELECT
	name,
	world_name,
	datacenter,
	high_quality,
	MIN(min_price) OVER (PARTITION BY name, world_name, high_quality) AS world_minimum_price,
	rank() OVER (PARTITION BY name, world_name, high_quality ORDER BY min_price) AS price_rank
FROM
	(SELECT
		name,
		world_name,
		datacenter,
		min_price,
		high_quality	
	FROM
		(SELECT
			items.name,
			price_world.world_name,
			price_world.datacenter,
			price_world.min_price,
			price_world.high_quality,
			rank() OVER (PARTITION BY price_world.update_time ORDER BY price_world.update_time DESC) AS recency_rank
		FROM
			items RIGHT JOIN (
				SELECT
					pl.item_id,
					worlds.name AS world_name,
					worlds.datacenter,
					pl.update_time,
					pl.min_price,
					pl.high_quality
				FROM
					(SELECT
						prices.price_id,
						prices.item_id,
						prices.world_id,
						prices.update_time,
						MIN(listings.price_per_unit) AS min_price,
						listings.high_quality
					FROM
						prices INNER JOIN listings USING (price_id)
					GROUP BY
						price_id,
						item_id,
						world_id,
						update_time,
						high_quality
					) pl INNER JOIN worlds USING (world_id)
			) price_world USING (item_id)
		WHERE
			%s
			AND items.marketable)
	WHERE
		recency_rank = 1))
WHERE price_rank = 1
GROUP BY
	name,
	world_name,
	datacenter,
	high_quality
ORDER BY
	high_quality DESC,
	datacenter,
	min_price;`, whereClause)
}

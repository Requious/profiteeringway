package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

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

func (p *Postgres) GetPriceForItemIDWorldSpecificExpensive(ctx context.Context, itemID int, worldName string) ([]*AllWorldsPriceRowExpensive, error) {
	query := recentAllWorldsPriceQueryExpensive("items.item_id = ($1) AND price_world.world_name = ($2)")
	rows, err := p.Db.QueryContext(ctx, query, itemID, worldName)
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

func recipeDetailsForItemID(itemID string) string {
	return fmt.Sprintf(`SELECT
	ing.*,
	items.name AS crafted_item_name
FROM (
SELECT	
	items.name AS ingredient_name,
	ingredients.ingredient_id,
	ingredients.ingredient_count,
	ingredients.crafted_item_id,
	ingredients.crafted_item_count AS crafted_quantity
FROM
	(SELECT
		r.crafted_item_id,
		r.crafted_item_count,
		r.ingredient_id,
		r.ingredient_count
	FROM
		items
			LEFT JOIN (
				SELECT
					recipes.crafted_item_id,
					recipes.crafted_item_count,
					recipe_ingredients.ingredient_id,
					recipe_ingredients.quantity AS ingredient_count
				FROM
					recipes INNER JOIN recipe_ingredients USING (recipe_id)
			) AS r ON items.item_id = r.crafted_item_id
		WHERE items.item_id = %v
	) AS ingredients INNER JOIN items ON ingredients.ingredient_id = items.item_id
) AS ing INNER JOIN items ON ing.crafted_item_id = items.item_id;
`, itemID)
}

type RecipeDetails struct {
	CraftedItemName  string
	CraftedItemCount int32
	CraftedItemID    int32
	Ingredients      []*Ingredient
}

type Ingredient struct {
	ItemID int32
	Name   string
	Count  int32
}

func (pg *Postgres) RecipesDetailsForItemID(ctx context.Context, itemID int32) (*RecipeDetails, error) {
	query := recipeDetailsForItemID("($1)")
	rows, err := pg.Db.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe details for item ID %v: %w", itemID, err)
	}

	details := &RecipeDetails{}
	initialized := false
	for rows.Next() {
		var craftedItemName, ingredientName string
		var craftedItemCount, craftedItemID, ingredientItemID, ingredientCount int32

		if err := rows.Scan(&ingredientName, &ingredientItemID, &ingredientCount, &craftedItemID, &craftedItemCount, &craftedItemName); err != nil {
			return nil, fmt.Errorf("failed to scan out values into row: %w", err)
		}

		if !initialized {
			details.CraftedItemName = craftedItemName
			details.CraftedItemCount = craftedItemCount
			details.CraftedItemID = craftedItemID
			initialized = true
		}

		ingredient := &Ingredient{
			ItemID: ingredientItemID,
			Name:   ingredientName,
			Count:  ingredientCount,
		}

		details.Ingredients = append(details.Ingredients, ingredient)
	}
	return details, nil
}

func (pg *Postgres) ConvertItemNameToItemID(ctx context.Context, itemName string) (int32, error) {
	row := pg.Db.QueryRowContext(ctx, `SELECT items.item_id FROM items WHERE items.name = ($1)`, itemName)
	var itemID int32
	if err := row.Scan(&itemID); err != nil {
		return 0, fmt.Errorf("failed to scan row value for item name lookup: %w", err)
	}

	return itemID, nil
}

func (pg *Postgres) WorldIDFromWorldName(ctx context.Context, worldName string) (int, error) {
	query := `SELECT
		world_id
	FROM
		worlds
	WHERE
		name = ($1)`

	row := pg.Db.QueryRowContext(ctx, query, worldName)
	var worldID int
	if err := row.Scan(&worldID); err != nil {
		return 0, fmt.Errorf("failed to get world ID for world: %s: %w", worldName, err)
	}
	return worldID, nil
}

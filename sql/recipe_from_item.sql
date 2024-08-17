SELECT
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
		WHERE items.item_id = 44178
	) AS ingredients INNER JOIN items ON ingredients.ingredient_id = items.item_id
) AS ing INNER JOIN items ON ing.crafted_item_id = items.item_id;

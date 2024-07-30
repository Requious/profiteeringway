SELECT
	items.item_id,
	items.name,
	named_recipe.ingredient_name,
	named_recipe.quantity
FROM
	items LEFT JOIN (
		SELECT
			recipes.crafted_item_id,
			ingredients.name AS ingredient_name,
			ingredients.quantity
		FROM
			recipes INNER JOIN (
				SELECT
					items.name,
					recipe_ingredients.quantity,
					recipe_ingredients.recipe_id
				FROM
					recipe_ingredients INNER JOIN items ON (recipe_ingredients.ingredient_id = items.item_id)
			) ingredients USING (recipe_id)
	) named_recipe ON (items.item_id = named_recipe.crafted_item_id)
WHERE
	items.name LIKE 'Archeo%'
ORDER BY
	items.item_id;

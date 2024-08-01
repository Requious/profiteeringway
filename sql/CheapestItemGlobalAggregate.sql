SELECT
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
			item_id = 44178
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
	min_price;

SELECT
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
	min_price_hq;

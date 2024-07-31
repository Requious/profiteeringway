SELECT
	name,
	world_name,
	min_price_hq
FROM
(SELECT
	items.name,
	price_world.world_name,
	price_world.min_price_hq,
	rank() OVER (PARTITION BY items.name ORDER BY price_world.min_price_hq) AS cheap_rank,
	rank() OVER (PARTITION BY price_world.update_time ORDER BY price_world.update_time DESC) AS recency_rank
FROM
	items RIGHT JOIN (
		SELECT
			prices.item_id,
			worlds.name AS world_name,
			prices.min_price_hq,
			prices.update_time
		FROM
			prices INNER JOIN worlds USING (world_id)
		WHERE
			prices.min_price_hq <> 0
	) price_world USING (item_id)
WHERE
	items.Type IN ('Marauder''s Arm','Two–handed Thaumaturge''s Arm','Weaver''s Primary Tool','Goldsmith''s Secondary Tool','Botanist''s Secondary Tool','Astrologian''s Arm','Fisher''s Primary Tool','Alchemist''s Primary Tool','Archer''s Arm','One–handed Conjurer''s Arm','Blacksmith''s Primary Tool','Arcanist''s Grimoire','Goldsmith''s Primary Tool','Alchemist''s Secondary Tool','Gladiator''s Arm','Red Mage''s Arm','Leatherworker''s Primary Tool','Scholar''s Arm','Earrings','Sage''s Arm','Blue Mage''s Arm','Rogue''s Arm','Blacksmith''s Secondary Tool','Culinarian''s Primary Tool','Reaper''s Arm','Miner''s Secondary Tool','Botanist''s Primary Tool','Culinarian''s Secondary Tool','Weaver''s Secondary Tool','Dancer''s Arm','Carpenter''s Secondary Tool','Armorer''s Primary Tool','Carpenter''s Primary Tool','Two–handed Conjurer''s Arm','Armorer''s Secondary Tool','One–handed Thaumaturge''s Arm','Dark Knight''s Arm','Miner''s Primary Tool','Samurai''s Arm','Shield','Fisher''s Secondary Tool','Machinist''s Arm','Hands','Body','Head','Necklace','Ring','Legs','Feet','Bracelets','Leatherworker''s Secondary Tool','Pugilist''s Arm','Lancer''s Arm','Pictomancer''s Arm','Viper''s Arm','Gunbreaker''s Arm')
	AND items.item_level > 700
	AND items.marketable)
WHERE
	cheap_rank < 4 AND
	recency_rank = 1;

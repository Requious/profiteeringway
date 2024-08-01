package postgres

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

func (p *Postgres) AllCrystals() ([]int, error) {
	return p.GetItemIDsForStaticQuery(`SELECT item_id FROM items WHERE type = 'Crystal';`)
}


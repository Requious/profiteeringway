CREATE OR REPLACE FUNCTION transfer_old_prices() RETURNS trigger AS $transfer_old_prices$
DECLARE
	price prices%ROWTYPE;
	listing listings%ROWTYPE;
	BEGIN
		FOR price IN
			SELECT
				*
			FROM
				prices	
			WHERE
				prices.item_id = NEW.item_id AND
				prices.world_id = NEW.world_id AND
				prices.price_id <> NEW.price_id
		LOOP
			-- price loop
			INSERT INTO prices_history SELECT price.*;

			FOR listing IN
				SELECT
					*
				FROM
					listings
				WHERE
					listings.price_id = price.price_id
			LOOP
				-- listing loop
				INSERT INTO listings_history SELECT listing.*;
			END LOOP;

			DELETE FROM listings WHERE listings.price_id = price.price_id;
			DELETE FROM prices WHERE prices.price_id = price.price_id;
		END LOOP;
		RETURN NEW;
	END;
$transfer_old_prices$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER transfer_old_prices AFTER INSERT ON prices
	FOR EACH ROW EXECUTE FUNCTION transfer_old_prices();

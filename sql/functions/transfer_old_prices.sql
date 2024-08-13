CREATE FUNCTION transfer_old_prices() RETURNS trigger AS $transfer_old_prices$
	BEGIN
	END;
$transfer_old_prices$ LANGUAGE plpgsql;

CREATE TRIGGER transfer_old_prices AFTER INSERT ON prices
	FOR EACH ROW EXECUTE FUNCTION transfer_old_prices();

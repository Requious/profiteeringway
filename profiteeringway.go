package main

import (
	"fmt"
	"profiteeringway/lib/postgres"
	"profiteeringway/secrets"
)

func main() {
	pg, err := postgres.NewPostgres(secrets.PostgresConnectionString)
	if err != nil {
		fmt.Printf("failed to initialize postgres: %v\n", err)
		return
	}

	item, err := pg.SelectItemWithID(31831)
	if err != nil {
		fmt.Printf("failed to retrieve item: %v\n", err)
		return
	}

	name, err := item.Name.Value()
	if err != nil {
		fmt.Printf("failed to scan name from item: %v\n", err)
	}

	fmt.Printf("got item %s", name)
}
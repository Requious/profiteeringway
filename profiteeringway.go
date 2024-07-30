package main

import (
	"fmt"
	"profiteeringway/lib/postgres"
	"profiteeringway/lib/universalis"
	"profiteeringway/secrets"
)

func main() {
	pg, err := postgres.NewPostgres(secrets.PostgresConnectionString)
	defer pg.CleanUp()

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

	if err = pg.InitializePriceTables(); err != nil {
		fmt.Printf("failed to initialize: %v", err)
	}

	priceData, err := universalis.GetItemData(54, []int{42892, 42893})
	if err != nil {
		fmt.Printf("failed to get Universalis data: %v", err)
		return
	}

	fmt.Printf("%+v\n", priceData)
}

package universalis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Universalis struct {
}

type UniversalisPriceData struct {
	Items map[string]struct {
		ItemID         int   `json:"itemID"`
		WorldID        int   `json:"worldID"`
		LastUploadTime int64 `json:"lastUploadTime"`
		Listings       []struct {
			PricePerUnit int  `json:"pricePerUnit"`
			Quantity     int  `json:"quantity"`
			Hq           bool `json:"hq"`
		} `json:"listings"`
		NqSaleVelocity int `json:"nqSaleVelocity"`
		HqSaleVelocity int `json:"hqSaleVelocity"`
		MinPriceNQ     int `json:"minPriceNQ"`
		MinPriceHQ     int `json:"minPriceHQ"`
	} `json:"items"`
}

func GetItemData() (*UniversalisPriceData, error) {
	resp, err := http.Get(`https://universalis.app/api/v2/57/44001,42458?fields=items.minPriceNQ,items.minPriceHQ,items.nqSaleVelocity,items.hqSaleVelocity, items.listings.pricePerUnit, items.listings.quantity,items.lastUploadTime,items.itemID,items.worldID, items.listings.hq&entriesWithin=86400&statsWithin=86400000`)
	if err != nil {
		return nil, fmt.Errorf("failed to get item from Universalis: %w", err)
	}

	priceData := &UniversalisPriceData{}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from Universalis: %w", err)
	}
	err = json.Unmarshal(body, priceData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json response from Universalis: %w", err)
	}
	return priceData, nil
}

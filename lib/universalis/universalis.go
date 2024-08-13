package universalis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const universalisBaseAPIUrl = "https://universalis.app/api/v2"

func fieldFilters() []string {
	return []string{
		"items.minPriceNQ",
		"items.minPriceHQ",
		"items.nqSaleVelocity",
		"items.hqSaleVelocity",
		"items.listings.pricePerUnit",
		"items.listings.quantity",
		"items.listings.hq",
		"items.lastUploadTime",
		"items.itemID",
		"items.worldID",
	}
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
		NqSaleVelocity float64 `json:"nqSaleVelocity"`
		HqSaleVelocity float64 `json:"hqSaleVelocity"`
		MinPriceNQ     int     `json:"minPriceNQ"`
		MinPriceHQ     int     `json:"minPriceHQ"`
	} `json:"items"`
}

func GetItemData(worldID int, itemIDs []int) (*UniversalisPriceData, error) {
	endpointUrl, err := url.Parse(universalisBaseAPIUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to build Universalis URL: %w", err)
	}

	var stringItemIDs []string

	for _, id := range itemIDs {
		stringItemIDs = append(stringItemIDs, strconv.Itoa(id))
	}

	itemIDsJoined := strings.Join(stringItemIDs, ",")

	endpointUrl = endpointUrl.JoinPath(strconv.Itoa(worldID), itemIDsJoined)
	q := endpointUrl.Query()
	q.Set("entriesWithin", "36000")
	q.Set("statsWithin", "36000000")
	q.Set("fields", strings.Join(fieldFilters(), ","))

	endpointUrl.RawQuery = q.Encode()

	finalizedUrl, err := url.PathUnescape(endpointUrl.String())
	if err != nil {
		return nil, fmt.Errorf("failed to unescape constructed url: %w", err)
	}

	resp, err := http.Get(finalizedUrl)
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
		return nil, fmt.Errorf("failed to unmarshal json response to %s from Universalis: %w", endpointUrl.String(), err)
	}
	return priceData, nil
}

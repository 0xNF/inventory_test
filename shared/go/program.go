package inventory_shared

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type InventoryProg struct {
	path string
}

func NewInventoryProg(path string) InventoryProg {
	return InventoryProg{
		path: path,
	}
}

func (p *InventoryProg) List(limit, offset *uint32, sortBy string, orderBy string, filter string, fields []string) ([]byte, error) {
	args := []string{"list", "--long", "--json"}

	if limit != nil {
		args = append(args, "--limit", fmt.Sprintf("%d", *limit))
	}
	if offset != nil {
		args = append(args, "--offset", fmt.Sprintf("%d", *offset))
	}
	if sortBy != "" {
		args = append(args, "--sort-by", sortBy)
	}
	if orderBy != "" {
		args = append(args, "--order-by", orderBy)
	}
	if filter != "" {
		args = append(args, "--filter", filter)
		if len(fields) > 0 {
			args = append(args, "--fields", strings.Join(fields, ","))
		}
	}

	cmd := exec.Command(p.path, args...)
	return cmd.Output()
}

func (p *InventoryProg) Add(itemData []byte) (InventoryItem, error) {
	cmd := exec.Command(p.path, "add", "--json", "--input", string(itemData))
	output, err := cmd.Output()
	if err != nil {
		return InventoryItem{}, fmt.Errorf("add item: failed to run command: %w", err)
	}
	var item InventoryItem
	err = json.Unmarshal(output, &item)
	if err != nil {
		return InventoryItem{}, fmt.Errorf("add item: failed to unmarshall json response: %w", err)
	}
	return item, nil
}

func (p *InventoryProg) Delete(id string) (GenericProgramResponse, error) {
	cmd := exec.Command(p.path, "remove", id)
	output, err := cmd.Output()
	if err != nil {
		return GenericProgramResponse{}, fmt.Errorf("delete item: failed to run delete command: %w", err)
	}
	var response GenericProgramResponse
	err = json.Unmarshal(output, &response)
	if err != nil {
		return GenericProgramResponse{}, fmt.Errorf("delete item: failed to unmarshal item delete response: %w", err)
	}
	return response, nil
}

func (p *InventoryProg) Edit(id string, itemData []byte) (GenericProgramResponse, error) {
	cmd := exec.Command(p.path, "edit", id, "--input", string(itemData), "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return GenericProgramResponse{}, fmt.Errorf("edit item: failed to run edit command: %w", err)
	}
	var response GenericProgramResponse
	err = json.Unmarshal(output, &response)
	if err != nil {
		return GenericProgramResponse{}, fmt.Errorf("edit item: failed to unmarshal item edit response: %w", err)
	}
	return response, nil
}

type PagingInfo struct {
	Limit  *uint32 `json:"limit,omitempty"`
	Offset *uint32 `json:"offset,omitempty"`
	Total  uint32  `json:"total"`
}

type PagedResponse struct {
	Items  []InventoryItem `json:"items"`
	Paging PagingInfo      `json:"paging"`
}

type InventoryItem struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	AcquiredDate      string `json:"acquired_date,omitempty"`
	PurchasePrice     int64  `json:"purchase_price,omitempty"`
	PurchaseCurrency  string `json:"purchase_currency,omitempty"`
	IsUsed            *bool  `json:"is_used,omitempty"`
	ReceivedFrom      string `json:"received_from,omitempty"`
	SerialNumber      string `json:"serial_number,omitempty"`
	PurchaseReference string `json:"purchase_reference,omitempty"`
	Notes             string `json:"notes,omitempty"`
	Extra             string `json:"extra,omitempty"`
	FuturePurchase    *bool  `json:"future_purchase,omitempty"`
}

type EditItemRequest struct {
	Name              string   `json:"name,omitempty"`
	AcquiredDate      string   `json:"acquired_date,omitempty"`
	PurchasePrice     *float64 `json:"purchase_price,omitempty"`
	PurchaseCurrency  string   `json:"purchase_currency,omitempty"`
	IsUsed            *bool    `json:"is_used,omitempty"`
	ReceivedFrom      string   `json:"received_from,omitempty"`
	SerialNumber      string   `json:"serial_number,omitempty"`
	PurchaseReference string   `json:"purchase_reference,omitempty"`
	Notes             string   `json:"notes,omitempty"`
	Extra             string   `json:"extra,omitempty"`
	FuturePurchase    *bool    `json:"future_purchase,omitempty"`
}

type GenericProgramResponse struct {
	Success  bool    `json:"success"`
	ItemId   string  `json:"item_id"`
	ItemName *string `json:"item_name"`
	Message  string  `json:"message"`
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

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

type PagingInfo struct {
	Limit  *uint32 `json:"limit,omitempty"`
	Offset *uint32 `json:"offset,omitempty"`
	Total  uint32  `json:"total"`
}

type PagedResponse struct {
	Items  []InventoryItem `json:"items"`
	Paging PagingInfo      `json:"paging"`
}

type InventoryProg struct {
	path string
}

type EditItemRequest struct {
	Name             string   `json:"name,omitempty"`
	AcquiredDate     string   `json:"acquired_date,omitempty"`
	PurchasePrice    float64  `json:"purchase_price,omitempty"`
	PurchaseCurrency string   `json:"purchase_currency,omitempty"`
	IsUsed           bool     `json:"is_used,omitempty"`
	ReceivedFrom     string   `json:"received_from,omitempty"`
	SerialNumber     string   `json:"serial_number,omitempty"`
	PurchaseReference string  `json:"purchase_reference,omitempty"`
	Notes            string   `json:"notes,omitempty"`
	Extra            string   `json:"extra,omitempty"`
	FuturePurchase   bool     `json:"future_purchase,omitempty"`
}

func (p *InventoryProg) list(limit, offset *uint32, sortBy string, orderBy string) ([]byte, error) {
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

	cmd := exec.Command(p.path, args...)
	return cmd.Output()
}

func (p *InventoryProg) add(itemData []byte) ([]byte, error) {
	cmd := exec.Command(p.path, "add", "--input", string(itemData))
	return cmd.Output()
}

func (p *InventoryProg) delete(id string) ([]byte, error) {
	cmd := exec.Command(p.path, "remove", id)
	return cmd.Output()
}

func (p *InventoryProg) edit(id string, itemData []byte) ([]byte, error) {
	cmd := exec.Command(p.path, "edit", id, "--input", string(itemData), "--json")
	return cmd.CombinedOutput()
}

var prog InventoryProg

func main() {
	prog = InventoryProg{
		path: "..\\inventory_manager_rs\\target\\debug\\inventory_manager_rs.exe",
	}

	// Serve static files from the static directory
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API endpoints
	http.HandleFunc("/api/items", handleItems)
	http.HandleFunc("/api/items/add", handleAddItem)
	http.HandleFunc("/api/items/remove", handleRemoveItem)
	http.HandleFunc("/api/items/edit/", handleEditItem)

	// Start the server
	fmt.Println("Starting server at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handleItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters for pagination and sorting
	query := r.URL.Query()
	var limit, offset *uint32

	if limitStr := query.Get("limit"); limitStr != "" {
		var l uint32
		fmt.Sscanf(limitStr, "%d", &l)
		limit = &l
	}
	if offsetStr := query.Get("offset"); offsetStr != "" {
		var o uint32
		fmt.Sscanf(offsetStr, "%d", &o)
		offset = &o
	}

	// Get sort parameters
	sortBy := query.Get("sortBy")
	orderBy := query.Get("orderBy")

	// Execute the inventory_manager_rs list command with pagination
	output, err := prog.list(limit, offset, sortBy, orderBy)
	if err != nil {
		http.Error(w, "Failed to execute inventory manager: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the content type header
	w.Header().Set("Content-Type", "application/json")

	// Write the raw output directly to the response
	w.Write(output)
}

func handleAddItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the JSON body
	var itemData json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&itemData); err != nil {
		http.Error(w, "Failed to parse JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Execute the inventory_manager_rs add command with JSON data
	output, err := prog.add(itemData)
	if err != nil {
		http.Error(w, "Failed to add item: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Item added successfully",
		"output":  string(output),
	})
}

func handleEditItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the item ID from the URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	itemID := parts[len(parts)-1]

	// Parse the request body
	var editReq EditItemRequest
	if err := json.NewDecoder(r.Body).Decode(&editReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert the edit request to JSON
	editJSON, err := json.Marshal(editReq)
	if err != nil {
		http.Error(w, "Error creating edit request", http.StatusInternalServerError)
		return
	}

	// Execute the edit command
	output, err := prog.edit(itemID, editJSON)
	if err != nil {
		log.Printf("Error executing edit command: %v\nOutput: %s", err, output)
		http.Error(w, "Error editing item", http.StatusInternalServerError)
		return
	}

	// Set response headers and return output
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

func handleRemoveItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON body
	var data struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Failed to parse JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate ID
	if data.ID == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Execute the inventory_manager_rs remove command
	output, err := prog.delete(data.ID)
	if err != nil {
		http.Error(w, "Failed to remove item: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Item removed successfully",
		"output":  string(output),
	})
}
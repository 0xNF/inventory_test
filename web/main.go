package main

import (
	"encoding/json"
	"fmt"
	"inventory_shared"
	"log"
	"net/http"
	"strings"
)

var prog inventory_shared.InventoryProg

func main() {
	prog = inventory_shared.NewInventoryProg("..\\rs\\target\\debug\\inventory_manager_rs.exe")

	// Serve static files from the static directory
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API endpoints
	http.HandleFunc("/api/items", handleItems)
	http.HandleFunc("/api/items/add", handleAddItem)
	http.HandleFunc("/api/items/remove", handleRemoveItem)
	http.HandleFunc("/api/items/edit/", handleEditItem)
	http.HandleFunc("/api/items/search", handleItems) // Search uses the same handler as it's just a filtered list

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

	// Get search parameters
	filter := query.Get("filter")
	fields := []string{}
	if fieldsStr := query.Get("fields"); fieldsStr != "" {
		fields = strings.Split(fieldsStr, ",")
	}

	// Execute the inventory_manager_rs list command with pagination and filtering
	output, err := prog.List(limit, offset, sortBy, orderBy, filter, fields)
	if err != nil {
		http.Error(w, "Failed to execute inventory manager: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the content type header

	// Write the raw output directly to the response
	bytes, err := json.Marshal(output)
	if err != nil {
		http.Error(w, "Failed to decode inventory manager data: "+err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func handleAddItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the JSON body
	var itemData inventory_shared.InventoryItem
	if err := json.NewDecoder(r.Body).Decode(&itemData); err != nil {
		http.Error(w, "Failed to parse JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Execute the inventory_manager_rs add command with JSON data
	output, err := prog.Add(itemData)
	if err != nil {
		http.Error(w, "Failed to add item: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Item added successfully",
		"output":  output,
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
	var editReq inventory_shared.EditItemRequest
	if err := json.NewDecoder(r.Body).Decode(&editReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Execute the edit command
	output, err := prog.Edit(itemID, editReq)
	if err != nil {
		log.Printf("Error executing edit command: Output: %s", err.Error())
		http.Error(w, "Error editing item", http.StatusInternalServerError)
		return
	}

	// Set response headers and return output
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Item edited successfully",
		"output":  output,
	})
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
	output, err := prog.Delete(data.ID)
	if err != nil {
		http.Error(w, "Failed to remove item: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Item removed successfully",
		"output":  output,
	})
}

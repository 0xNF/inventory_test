# Inventory Manager Web

A web-based frontend for the inventory_manager_rs CLI tool.

## Features

- List all inventory items with sorting and filtering
- Add new inventory items through a user-friendly interface
- Remove inventory items with confirmation
- Interactive UI with notifications and responsive design

## Requirements

- Go 1.21 or later
- The `inventory_manager_rs` CLI tool must be in your PATH

## Setup

1. Make sure the `inventory_manager_rs` CLI tool is compiled and available in your PATH
2. Clone this repository
3. Navigate to the project directory
4. Run `go run main.go`
5. Open your browser and navigate to http://localhost:8080

## Usage

### Viewing Inventory

The main page displays all inventory items in a table. You can:
- Sort by name, date purchased, or price (ascending or descending)
- Filter items by typing in the search box
- Delete items using the delete button

### Adding Items

Fill out the form at the top of the page to add a new item:
- "Item Name" is required
- Other fields are optional
- Click "Add Item" to submit

## Architecture

This project follows a simple architecture:

- Backend: Go HTTP server that wraps calls to the inventory_manager_rs CLI
- Frontend: HTML5, CSS, and vanilla JavaScript for a responsive UI
- Communication: RESTful API endpoints for CRUD operations

## API Endpoints

- `GET /api/items` - Get all inventory items
- `POST /api/items/add` - Add a new inventory item
- `POST /api/items/remove` - Remove an inventory item

## License

This project is licensed under the MIT License.
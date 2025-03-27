document.addEventListener('DOMContentLoaded', function() {
    // Tab switching functionality
    const tabButtons = document.querySelectorAll('.tab-button');
    const tabContents = document.querySelectorAll('.tab-content');

    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            // Remove active class from all buttons and sections
            tabButtons.forEach(btn => btn.classList.remove('active'));
            tabContents.forEach(content => content.style.display = 'none');
            
            // Add active class to clicked button
            button.classList.add('active');
            
            // Show corresponding content
            const tabId = button.getAttribute('data-tab');
            document.getElementById(tabId + '-section').style.display = 'block';
        });
    });

    // Show default tab (list)
    document.querySelector('.tab-button[data-tab="list"]').click();

    // DOM elements
    const inventoryTable = document.getElementById('inventory-items');
    const addItemForm = document.getElementById('add-item-form');
    const searchInput = document.getElementById('search');
    const sortBySelect = document.getElementById('sort-by');
    const sortAscButton = document.getElementById('sort-asc');
    const sortDescButton = document.getElementById('sort-desc');
    const prevPageButton = document.getElementById('prev-page');
    const nextPageButton = document.getElementById('next-page');
    const pageInfo = document.getElementById('page-info');
    const itemsPerPageSelect = document.getElementById('items-per-page');
    
    // State variables
    let inventoryItems = [];
    let sortField = 'name';
    let sortDirection = 'asc';
    let currentPage = 1;
    let itemsPerPage = 10;
    let totalItems = 0;
    
    // Load inventory items on page load
    loadInventoryItems();
    
    // Setup event listeners
    addItemForm.addEventListener('submit', handleAddItem);
    searchInput.addEventListener('input', filterItems);
    sortBySelect.addEventListener('change', handleSortChange);
    sortAscButton.addEventListener('click', () => setSortDirection('asc'));
    sortDescButton.addEventListener('click', () => setSortDirection('desc'));
    prevPageButton.addEventListener('click', () => changePage(-1));
    nextPageButton.addEventListener('click', () => changePage(1));
    itemsPerPageSelect.addEventListener('change', handleItemsPerPageChange);
    
    // Functions
    function loadInventoryItems() {
        const offset = (currentPage - 1) * itemsPerPage;
        fetch(`/api/items?limit=${itemsPerPage}&offset=${offset}&sortBy=${sortField}&orderBy=${sortDirection}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to fetch inventory items');
                }
                return response.json();
            })
            .then(data => {
                if (!data || !data.items) {
                    throw new Error('Invalid response format');
                }
                inventoryItems = data.items;
                totalItems = data.paging.total;
                updatePaginationControls();
                updateSortButtons();
                renderInventoryItems();
            })
            .catch(error => {
                console.error('Error loading inventory items:', error);
                showNotification('Error loading inventory items', 'error');
            });
    }
    
    function updatePaginationControls() {
        const totalPages = Math.ceil(totalItems / itemsPerPage);
        prevPageButton.disabled = currentPage === 1;
        nextPageButton.disabled = currentPage === totalPages;
        pageInfo.textContent = `Page ${currentPage} of ${totalPages} (${totalItems} items)`;
    }
    
    function changePage(delta) {
        currentPage += delta;
        loadInventoryItems();
    }
    
    function handleItemsPerPageChange(event) {
        itemsPerPage = parseInt(event.target.value);
        currentPage = 1; // Reset to first page
        loadInventoryItems();
    }
    
    function renderInventoryItems() {
        // Clear the table
        inventoryTable.innerHTML = '';
        
        // Check if inventoryItems exists
        if (!inventoryItems || !Array.isArray(inventoryItems)) {
            console.error('Invalid inventory items data');
            return;
        }
        
        // Render each item
        inventoryItems.forEach(item => {
            const row = document.createElement('tr');
            
            // Format the date
            const purchaseDate = item.acquired_date ? new Date(item.acquired_date).toLocaleDateString() : 'N/A';
            
            // Format the price with proper currency handling
            const price = item.purchase_price ?
                new Intl.NumberFormat('en-US', {
                    style: 'currency',
                    currency: item.purchase_currency && item.purchase_currency.trim() !== '' ? item.purchase_currency : 'JPY'
                }).format(item.purchase_price) :
                'N/A';
            
            row.innerHTML = `
                <td>${item.id}</td>
                <td>${item.name}</td>
                <td>${purchaseDate}</td>
                <td>${price}</td>
                <td>${item.is_used ? 'Yes' : 'No'}</td>
                <td>${item.future_purchase ? 'Yes' : 'No'}</td>
                <td>
                    <button class="action-btn delete-btn" data-id="${item.id}">Delete</button>
                </td>
            `;
            
            inventoryTable.appendChild(row);
        });
        
        // Add event listeners to delete buttons
        document.querySelectorAll('.delete-btn').forEach(button => {
            button.addEventListener('click', handleDeleteItem);
        });
    }
    
    function handleAddItem(event) {
        event.preventDefault();
        
        // Create JSON object from form data
        const itemData = {
            name: document.getElementById('item-name').value,
            acquired_date: document.getElementById('date-purchased').value || null,
            purchase_price: parseFloat(document.getElementById('purchase-price').value) || null,
            purchase_currency: document.getElementById('purchase-currency').value || 'USD',
            purchase_reference: document.getElementById('purchase-ref').value || null,
            is_used: document.getElementById('is-used').checked,
            future_purchase: document.getElementById('future-purchase').checked,
            notes: document.getElementById('notes').value || null
        };
        
        // Send data to server
        fetch('/api/items/add', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(itemData)
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to add item');
            }
            return response.json();
        })
        .then(data => {
            console.log('Item added successfully:', data);
            showNotification('Item added successfully', 'success');
            
            // Reset form and reload items
            addItemForm.reset();
            loadInventoryItems();
        })
        .catch(error => {
            console.error('Error adding item:', error);
            showNotification('Error adding item', 'error');
        });
    }
    
    function handleDeleteItem(event) {
        const id = event.target.getAttribute('data-id');
        
        if (confirm('Are you sure you want to delete this item?')) {
            fetch('/api/items/remove', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ id: id })
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to delete item');
                }
                return response.json();
            })
            .then(data => {
                console.log('Item deleted successfully:', data);
                showNotification('Item deleted successfully', 'success');
                loadInventoryItems();
            })
            .catch(error => {
                console.error('Error deleting item:', error);
                showNotification('Error deleting item', 'error');
            });
        }
    }
    
    function filterItems() {
        const filterText = searchInput.value.toLowerCase();
        currentPage = 1; // Reset to first page when filtering
        loadInventoryItems();
    }
    
    function handleSortChange() {
        sortField = sortBySelect.value;
        currentPage = 1; // Reset to first page when sorting
        loadInventoryItems();
    }
    
    function setSortDirection(direction) {
        sortDirection = direction;
        updateSortButtons();
        currentPage = 1; // Reset to first page when changing sort direction
        loadInventoryItems();
    }

    function updateSortButtons() {
        // Update active class on buttons based on current sortDirection
        if (sortDirection === 'asc') {
            sortAscButton.classList.add('active');
            sortDescButton.classList.remove('active');
        } else {
            sortAscButton.classList.remove('active');
            sortDescButton.classList.add('active');
        }
    }
    
    function showNotification(message, type) {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        
        // Add to the document
        document.body.appendChild(notification);
        
        // Remove after a delay
        setTimeout(() => {
            notification.remove();
        }, 3000);
    }
});
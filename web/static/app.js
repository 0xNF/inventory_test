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
    let searchTerm = '';
    
    // Load inventory items on page load
    loadInventoryItems();
    
    // Setup event listeners
    addItemForm.addEventListener('submit', handleAddItem);
    searchInput.addEventListener('keyup', function(event) {
        if (event.key === 'Enter') {
            filterItems();
        }
    });
    searchInput.addEventListener('input', function() {
        // Add debounce for live search
        clearTimeout(searchInput.debounceTimer);
        searchInput.debounceTimer = setTimeout(filterItems, 500);
    });
    sortBySelect.addEventListener('change', handleSortChange);
    sortAscButton.addEventListener('click', () => setSortDirection('asc'));
    sortDescButton.addEventListener('click', () => setSortDirection('desc'));
    prevPageButton.addEventListener('click', () => changePage(-1));
    nextPageButton.addEventListener('click', () => changePage(1));
    itemsPerPageSelect.addEventListener('change', handleItemsPerPageChange);
    
    // Functions
    function loadInventoryItems() {
        const offset = (currentPage - 1) * itemsPerPage;
        let url = `/api/items?limit=${itemsPerPage}&offset=${offset}&sortBy=${sortField}&orderBy=${sortDirection}`;
        
        // Add search term if it exists
        if (searchTerm) {
            url += `&filter=${encodeURIComponent(searchTerm)}`;
        }
        
        fetch(url)
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
        const navigationButtons = document.querySelectorAll('.pagination-button');
        const pageInfoSpan = document.getElementById('page-info');
        
        // Only hide navigation buttons and page info if all items fit on one page
        if (totalItems <= itemsPerPage) {
            navigationButtons.forEach(btn => btn.style.display = 'none');
            pageInfoSpan.style.display = 'none';
        } else {
            navigationButtons.forEach(btn => btn.style.display = 'inline-block');
            pageInfoSpan.style.display = 'inline';
            prevPageButton.disabled = currentPage === 1;
            nextPageButton.disabled = currentPage === totalPages;
            pageInfo.textContent = `Page ${currentPage} of ${totalPages} (${totalItems} items)`;
        }
        
        // Items per page dropdown should always be visible
        document.getElementById('items-per-page').style.display = 'inline-block';
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
    
    function setupInfoIconListeners() {
    const modal = document.getElementById('info-modal');
    const span = document.getElementsByClassName('close')[0];
    
    // Add click event to all info icons
    document.querySelectorAll('.info-icon').forEach(icon => {
        icon.addEventListener('click', (e) => {
            const itemId = e.target.getAttribute('data-id');
            const item = inventoryItems.find(item => item.id === itemId);
            
            if (item) {
                document.getElementById('modal-purchase-ref').textContent = item.purchase_reference || 'N/A';
                document.getElementById('modal-received-from').textContent = item.received_from || 'N/A';
                document.getElementById('modal-serial-number').textContent = item.serial_number || 'N/A';
                modal.style.display = 'block';
            }
        });
    });
    
    // Close modal when clicking (x)
    span.onclick = function() {
        modal.style.display = 'none';
    }
    
    // Close modal when clicking outside
    window.onclick = function(event) {
        if (event.target == modal) {
            modal.style.display = 'none';
        }
    }
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
                <td>
                    <span class="info-icon" data-id="${item.id}">ℹ️</span>
                    ${item.id}
                </td>
                <td>${item.name}</td>
                <td>${purchaseDate}</td>
                <td>${price}</td>
                <td>${item.is_used ? 'Yes' : 'No'}</td>
                <td>${item.future_purchase ? 'Yes' : 'No'}</td>
                <td>
                <button class="action-btn edit-btn" data-id="${item.id}">Edit</button>
                </td>
            `;
            
            inventoryTable.appendChild(row);
        });
        
        // Add event listeners to delete buttons and info icons
        document.querySelectorAll('.edit-btn').forEach(button => {
            button.addEventListener('click', handleEditItem);
        });

    // Add info icon listeners
    document.querySelectorAll('.info-icon').forEach(icon => {
        icon.addEventListener('click', (e) => {
            const itemId = e.target.getAttribute('data-id');
            const item = inventoryItems.find(item => item.id === itemId);
            
            if (item) {
                // Format date and price for display
                const purchaseDate = item.acquired_date ? new Date(item.acquired_date).toLocaleDateString() : 'N/A';
                const price = item.purchase_price ?
                    new Intl.NumberFormat('en-US', {
                        style: 'currency',
                        currency: item.purchase_currency || 'JPY'
                    }).format(item.purchase_price) : 'N/A';

                // Update all modal fields
                document.getElementById('modal-id').textContent = item.id;
                document.getElementById('modal-name').textContent = item.name;
                document.getElementById('modal-acquired-date').textContent = purchaseDate;
                document.getElementById('modal-purchase-price').textContent = price;
                document.getElementById('modal-purchase-currency').textContent = item.purchase_currency || 'N/A';
                document.getElementById('modal-purchase-ref').textContent = item.purchase_reference || 'N/A';
                document.getElementById('modal-received-from').textContent = item.received_from || 'N/A';
                document.getElementById('modal-serial-number').textContent = item.serial_number || 'N/A';
                document.getElementById('modal-is-used').textContent = item.is_used ? 'Yes' : 'No';
                document.getElementById('modal-future-purchase').textContent = item.future_purchase ? 'Yes' : 'No';
                document.getElementById('modal-notes').textContent = item.notes || 'N/A';

                document.getElementById('info-modal').style.display = 'block';
            }
        });
    });

    // Add modal close button listener
    const modal = document.getElementById('info-modal');
    const closeBtn = modal.querySelector('.close');
    if (closeBtn) {
        closeBtn.addEventListener('click', () => {
            modal.style.display = 'none';
        });
    }

    // Close modal when clicking outside or pressing Escape
    window.addEventListener('click', (event) => {
        if (event.target === modal) {
            modal.style.display = 'none';
        }
    });

    window.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && modal.style.display === 'block') {
            modal.style.display = 'none';
        }
    });

        // Setup info icon listeners
        setupInfoIconListeners();
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
    
    function handleEditItem(event) {
        const id = event.target.getAttribute('data-id');
        const item = inventoryItems.find(item => item.id === id);
        
        if (item) {
            // Populate the edit form
            document.getElementById('edit-item-id').value = item.id;
            document.getElementById('edit-item-name').value = item.name || '';
            document.getElementById('edit-date-purchased').value = item.acquired_date || '';
            document.getElementById('edit-purchase-price').value = item.purchase_price || '';
            document.getElementById('edit-purchase-currency').value = item.purchase_currency || 'USD';
            document.getElementById('edit-purchase-ref').value = item.purchase_reference || '';
            document.getElementById('edit-is-used').checked = item.is_used || false;
            document.getElementById('edit-future-purchase').checked = item.future_purchase || false;
            document.getElementById('edit-notes').value = item.notes || '';
            
            // Show the modal
            document.getElementById('edit-modal').style.display = 'block';
        }
    }

    // Add edit form submit handler
    document.getElementById('edit-item-form').addEventListener('submit', handleEditFormSubmit);

    // Add edit modal close button handler
    const editModal = document.getElementById('edit-modal');
    const editCloseBtn = editModal.querySelector('.close');
    if (editCloseBtn) {
        editCloseBtn.addEventListener('click', () => {
            editModal.style.display = 'none';
        });
    }

    // Close edit modal when clicking outside
    window.addEventListener('click', (event) => {
        if (event.target === editModal) {
            editModal.style.display = 'none';
        }
    });

    function handleEditFormSubmit(event) {
        event.preventDefault();
        
        const id = document.getElementById('edit-item-id').value;
        
        // Create JSON object from form data
        const itemData = {
            name: document.getElementById('edit-item-name').value,
            acquired_date: document.getElementById('edit-date-purchased').value || null,
            purchase_price: parseFloat(document.getElementById('edit-purchase-price').value) || null,
            purchase_currency: document.getElementById('edit-purchase-currency').value || null,
            purchase_reference: document.getElementById('edit-purchase-ref').value || null,
            is_used: document.getElementById('edit-is-used').checked,
            future_purchase: document.getElementById('edit-future-purchase').checked,
            notes: document.getElementById('edit-notes').value || null
        };
        
        // Send data to server
        fetch(`/api/items/edit/${id}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(itemData)
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to edit item');
            }
            return response.json();
        })
        .then(data => {
            showNotification('Item updated successfully', 'success');
            document.getElementById('edit-modal').style.display = 'none';
            loadInventoryItems();
        })
        .catch(error => {
            console.error('Error editing item:', error);
            showNotification('Error updating item', 'error');
        });
    }
    
    function filterItems() {
        searchTerm = searchInput.value.trim();
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
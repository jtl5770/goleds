document.addEventListener('DOMContentLoaded', () => {
    const formContainer = document.getElementById('config-form');
    const saveButton = document.getElementById('save-button');
    const messageDiv = document.getElementById('message');
    let originalConfig = null;

    // Fetch config and build the form
    fetch('/api/config')
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        })
        .then(config => {
            originalConfig = config;
            buildForm(originalConfig, formContainer);
        })
        .catch(error => {
            messageDiv.textContent = `Error loading configuration: ${error}`;
            messageDiv.style.color = 'red';
        });

    // Handle form submission
    saveButton.addEventListener('click', () => {
        if (!originalConfig) {
            messageDiv.textContent = 'Configuration not loaded yet. Cannot save.';
            messageDiv.style.color = 'red';
            return;
        }

        const updatedConfig = updateConfigFromForm(originalConfig, formContainer);
        
        fetch('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(updatedConfig),
        })
        .then(response => {
            if (!response.ok) {
                return response.text().then(text => { throw new Error(text) });
            }
            return response.text();
        })
        .then(data => {
            messageDiv.textContent = 'Configuration saved successfully! The application will now reload.';
            messageDiv.style.color = 'green';
            setTimeout(() => { messageDiv.textContent = ''; }, 5000);
        })
        .catch(error => {
            messageDiv.textContent = `Error saving configuration: ${error}`;
            messageDiv.style.color = 'red';
        });
    });
});

// --- Form Building ---

const isColorField = (key) => /LedRGB$|^Led(Hour|Minute|Green|Yellow|Red)$/i.test(key);
const isDurationField = (key) => /Duration$|Delay$|Time$|UpdateFreq$/i.test(key);

function buildForm(config, container) {
    container.innerHTML = '';
    buildRecursive(config, container, []);
}

function buildRecursive(data, parentElement, path) {
    for (const key in data) {
        const value = data[key];
        const currentPath = [...path, key];
        const pathString = currentPath.join('.');

        if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
            const fieldset = document.createElement('fieldset');
            const legend = document.createElement('legend');
            legend.textContent = key;
            fieldset.appendChild(legend);
            parentElement.appendChild(fieldset);
            buildRecursive(value, fieldset, currentPath);
        } else {
            const div = document.createElement('div');
            div.className = 'form-control';
            const label = document.createElement('label');
            label.textContent = key;
            label.htmlFor = pathString;
            div.appendChild(label);

            // Special handling for NightLED.LedRGB
            if (pathString === 'NightLED.LedRGB' && Array.isArray(value)) {
                const listContainer = document.createElement('div');
                listContainer.className = 'color-list-container';
                listContainer.dataset.path = pathString;

                value.forEach(color => {
                    listContainer.appendChild(createColorListItem(color));
                });

                const addButton = document.createElement('button');
                addButton.textContent = 'Add Color';
                addButton.className = 'add-color-btn';
                addButton.type = 'button';
                addButton.addEventListener('click', () => {
                    listContainer.insertBefore(createColorListItem([0, 0, 0]),listContainer.lastChild);
                });
                
                div.appendChild(listContainer);
                listContainer.appendChild(addButton);
                
                // Init drag and drop
                let draggedItem = null;
                listContainer.addEventListener('dragstart', e => {
                    draggedItem = e.target.closest('.color-list-item');
                    setTimeout(() => {
                        if (draggedItem) draggedItem.style.display = 'none';
                    }, 0);
                });

                listContainer.addEventListener('dragend', e => {
                    setTimeout(() => {
                        if (draggedItem) {
                            draggedItem.style.display = 'flex';
                            draggedItem = null;
                        }
                    }, 0);
                });

                listContainer.addEventListener('dragover', e => {
                    e.preventDefault();
                    const afterElement = getDragAfterElement(listContainer, e.clientY);
                    const currentItem = document.querySelector('.dragging');
                    if (afterElement == null || afterElement == listContainer.lastChild) {
                        listContainer.insertBefore(draggedItem, listContainer.lastChild);
                    } else {
                        listContainer.insertBefore(draggedItem, afterElement);
                    }
                });

            } else if (pathString === 'MultiBlobLED.BlobCfg' && Array.isArray(value)) {
                const listContainer = document.createElement('div');
                listContainer.className = 'blob-list-container';
                listContainer.dataset.path = pathString;

                value.forEach(blob => {
                    listContainer.appendChild(createBlobListItem(blob));
                });

                const addButton = document.createElement('button');
                addButton.textContent = 'Add Blob';
                addButton.className = 'add-blob-btn';
                addButton.type = 'button';
                addButton.addEventListener('click', () => {
                    listContainer.insertBefore(createBlobListItem({ DeltaX: 0.1, X: 50, Width: 512, LedRGB: [255,0,0]}),
                        listContainer.lastChild);
                });
                listContainer.appendChild(addButton);
                div.appendChild(listContainer);
            } else if (isDurationField(key)) {
                const container = document.createElement('div');
                container.className = 'duration-input-container';
                const input = document.createElement('input');
                input.id = pathString;
                input.dataset.path = pathString;
                input.dataset.type = 'duration';
                input.type = 'number';
                input.value = value / 1000000; // Convert ns to ms
                container.appendChild(input);
                const unitLabel = document.createElement('span');
                unitLabel.textContent = 'ms';
                container.appendChild(unitLabel);
                div.appendChild(container);
            } else if (isColorField(key) && Array.isArray(value)) {
                const container = document.createElement('div');
                container.className = 'rgb-input-container';
                container.dataset.path = pathString;
                const [r, g, b] = value;
                container.appendChild(createLabeledInput('R', r));
                container.appendChild(createLabeledInput('G', g));
                container.appendChild(createLabeledInput('B', b));
                div.appendChild(container);
            } else {
                const input = document.createElement('input');
                input.id = pathString;
                input.dataset.path = pathString;
                input.dataset.type = Array.isArray(value) ? 'array' : typeof value;
                if (typeof value === 'boolean') {
                    input.type = 'checkbox';
                    input.checked = value;
                } else {
                    input.type = 'text';
                    input.value = Array.isArray(value) ? JSON.stringify(value) : value;
                }
                div.appendChild(input);
            }
            parentElement.appendChild(div);
        }
    }
}

function getDragAfterElement(container, y) {
    const draggableElements = [...container.querySelectorAll('.color-list-item:not(.dragging)')];

    return draggableElements.reduce((closest, child) => {
        const box = child.getBoundingClientRect();
        const offset = y - box.top - box.height / 2;
        if (offset < 0 && offset > closest.offset) {
            return { offset: offset, element: child };
        } else {
            return closest;
        }
    }, { offset: Number.NEGATIVE_INFINITY }).element;
}


function createColorListItem(color) {
    const item = document.createElement('div');
    item.className = 'color-list-item';
    item.draggable = true;

    const handle = document.createElement('span');
    handle.className = 'drag-handle';
    handle.textContent = '☰';
    item.appendChild(handle);

    const [r, g, b] = color;
    item.appendChild(createLabeledInput('R', r));
    item.appendChild(createLabeledInput('G', g));
    item.appendChild(createLabeledInput('B', b));

    const deleteBtn = document.createElement('span');
    deleteBtn.className = 'delete-color-btn';
    deleteBtn.textContent = '✖';
    deleteBtn.addEventListener('click', () => {
        // Ensure at least one item remains
        if (item.parentElement.children.length > 1) {
            item.remove();
        } else {
            alert('At least one color is required.');
        }
    });
    item.appendChild(deleteBtn);

    return item;
}

function createBlobListItem(blob) {
    const item = document.createElement('div');
    item.className = 'blob-list-item';

    // Create inputs for blob properties
    item.appendChild(createLabeledInput('DeltaX', blob.DeltaX, -1, 1, 0.1));
    item.appendChild(createLabeledInput('X', blob.X, 0, 1000, 1));
    item.appendChild(createLabeledInput('Width', blob.Width, 0, 1000, 1));

    const [r, g, b] = blob.LedRGB;
    const rgbContainer = document.createElement('div');
    rgbContainer.className = 'rgb-input-container';
    rgbContainer.appendChild(createLabeledInput('R', r));
    rgbContainer.appendChild(createLabeledInput('G', g));
    rgbContainer.appendChild(createLabeledInput('B', b));
    item.appendChild(rgbContainer);

    const deleteBtn = document.createElement('span');
    deleteBtn.className = 'delete-blob-btn';
    deleteBtn.textContent = '✖';
    deleteBtn.addEventListener('click', () => {
        item.remove();
    });
    item.appendChild(deleteBtn);

    return item;
}

function createNumberInput(value) {
    const input = document.createElement('input');
    input.type = 'number';
    input.min = 0;
    input.max = 255;
    input.step = 1;
    input.value = value;
    return input;
}

function createLabeledInput(labelText, value, min = 0, max = 255, step = 1) {
    const container = document.createElement('div');
    container.className = 'labeled-input';
    const label = document.createElement('label');
    label.textContent = labelText;
    container.appendChild(label);
    container.appendChild(createNumberInput(value, min, max, step));
    return container;
}

// --- Form Data Handling ---

function updateConfigFromForm(config, container) {
    // Handle standard inputs
    const inputs = container.querySelectorAll('input[data-path]:not([type="number"]), textarea[data-path]');
    inputs.forEach(input => {
        const path = input.dataset.path;
        if (!path) return;
        
        const pathParts = path.split('.');
        let value;
        if (input.tagName === 'TEXTAREA') {
            try { value = JSON.parse(input.value); } catch (e) { value = []; }
        } else if (input.type === 'checkbox') {
            value = input.checked;
        } else {
            value = input.value;
            if (input.dataset.type === 'duration') {
                value = `${value}ms`;
            } else if (input.dataset.type === 'number') {
                value = parseFloat(value);
            } else if (input.dataset.type === 'array') {
                try { value = JSON.parse(value); } catch (e) {}
            }
        }
        
        let current = config;
        for (let i = 0; i < pathParts.length - 1; i++) {
            current = current[pathParts[i]];
        }
        current[pathParts[pathParts.length - 1]] = value;
    });

    // Handle single RGB color inputs
    const pickers = container.querySelectorAll('.rgb-input-container');
    pickers.forEach(picker => {
        const path = picker.dataset.path;
        const numberInputs = picker.querySelectorAll('input[type="number"]');
        const r = parseInt(numberInputs[0].value, 10) || 0;
        const g = parseInt(numberInputs[1].value, 10) || 0;
        const b = parseInt(numberInputs[2].value, 10) || 0;
        const finalRgb = [r, g, b];
        
        const pathParts = path.split('.');
        let current = config;
        for (let i = 0; i < pathParts.length - 1; i++) {
            current = current[pathParts[i]];
        }
        current[pathParts[pathParts.length - 1]] = finalRgb;
    });

    // Handle NightLED color list
    const nightLedList = container.querySelector('.color-list-container[data-path="NightLED.LedRGB"]');
    if (nightLedList) {
        const colors = [];
        const items = nightLedList.querySelectorAll('.color-list-item');
        items.forEach(item => {
            const numberInputs = item.querySelectorAll('input[type="number"]');
            const r = parseInt(numberInputs[0].value, 10) || 0;
            const g = parseInt(numberInputs[1].value, 10) || 0;
            const b = parseInt(numberInputs[2].value, 10) || 0;
            colors.push([r, g, b]);
        });
        config.NightLED.LedRGB = colors;
    }

    // Handle MultiBlobLED blob list
    const blobList = container.querySelector('.blob-list-container[data-path="MultiBlobLED.BlobCfg"]');
    if (blobList) {
        const blobs = [];
        const items = blobList.querySelectorAll('.blob-list-item');
        items.forEach(item => {
            const numberInputs = item.querySelectorAll('input[type="number"]');
            const deltaX = parseFloat(numberInputs[0].value) || 0;
            const x = parseInt(numberInputs[1].value, 10) || 0;
            const width = parseInt(numberInputs[2].value, 10) || 0;
            const r = parseInt(numberInputs[3].value, 10) || 0;
            const g = parseInt(numberInputs[4].value, 10) || 0;
            const b = parseInt(numberInputs[5].value, 10) || 0;
            blobs.push({ DeltaX: deltaX, X: x, Width: width, LedRGB: [r, g, b] });
        });
        config.MultiBlobLED.BlobCfg = blobs;
    }

    return config;
}

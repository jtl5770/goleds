document.addEventListener('DOMContentLoaded', () => {
    const formContainer = document.getElementById('config-form');
    const saveButton = document.getElementById('save-button');
    const resetButton = document.getElementById('reset-button');
    const messageDiv = document.getElementById('message');
    let originalConfig = null;

    function repositionMessageDiv() {
        const formRect = formContainer.getBoundingClientRect();
        messageDiv.style.left = `${formRect.left + formRect.width / 2}px`;
        messageDiv.style.transform = 'translateX(-50%)';
    }

    function showMessage(message, type = 'info', duration = 3000) {
        repositionMessageDiv();
        messageDiv.textContent = message;
        messageDiv.className = `show ${type}`;
        setTimeout(() => {
            messageDiv.className = messageDiv.className.replace('show', '');
        }, duration);
    }

    window.addEventListener('resize', repositionMessageDiv);

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
            showMessage(`Error loading configuration: ${error}`, 'error');
        });

    // Handle form submission
    saveButton.addEventListener('click', () => {
        if (!originalConfig) {
            showMessage('Configuration not loaded yet. Cannot save.', 'error');
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
            showMessage('Configuration saved successfully! The application will now reload.', 'success', 5000);
        })
        .catch(error => {
            showMessage(`Error saving configuration: ${error}`, 'error');
        });
    });

    resetButton.addEventListener('click', () => {
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
                showMessage('Form has been reset with the latest configuration from the server.', 'info');
            })
            .catch(error => {
                showMessage(`Error loading configuration: ${error}`, 'error');
            });
    });
});

// --- Form Building ---

const isColorField = (key) => /LedRGB$|^Led(Hour|Minute|Green|Yellow|Red)$/i.test(key);
const isDurationField = (key) => /(Duration|Delay|Time|UpdateFreq)$/i.test(key);
const isNonNegativeNumber = (key) => /^(SampleRate|FramesPerBuffer|Width)$/i.test(key);
const isDbField = (key) => /^(MinDB|MaxDB)$/i.test(key);
const isLedPosition = (key) => /^(StartLed|EndLed)/i.test(key);

function buildForm(config, container) {
    container.innerHTML = '';
    buildRecursive(config, container, [], config.LedsTotal);
}

function buildRecursive(data, parentElement, path, ledsTotal) {
    for (const key in data) {
        if (key === 'LedsTotal') continue; // Don't show this field
        const value = data[key];
        const currentPath = [...path, key];
        const pathString = currentPath.join('.');

        if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
            const fieldset = document.createElement('fieldset');
            const legend = document.createElement('legend');
            legend.textContent = key;
            fieldset.appendChild(legend);
            parentElement.appendChild(fieldset);
            buildRecursive(value, fieldset, currentPath, ledsTotal);
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
                    const newItem = createColorListItem([0, 0, 0]);
                    listContainer.insertBefore(newItem, addButton);
                    updateArrowStates(listContainer);
                });
                
                div.appendChild(listContainer);
                listContainer.appendChild(addButton);
                
                updateArrowStates(listContainer);

            } else if (pathString === 'MultiBlobLED.BlobCfg' && Array.isArray(value)) {
                const listContainer = document.createElement('div');
                listContainer.className = 'blob-list-container';
                listContainer.dataset.path = pathString;

                value.forEach(blob => {
                    listContainer.appendChild(createBlobListItem(blob, ledsTotal));
                });

                const addButton = document.createElement('button');
                addButton.textContent = 'Add Blob';
                addButton.className = 'add-blob-btn';
                addButton.type = 'button';
                addButton.addEventListener('click', () => {
                    const newItem = createBlobListItem({ DeltaX: 0.1, X: 50, Width: 512, LedRGB: [255,0,0]}, ledsTotal);
                    listContainer.insertBefore(newItem, addButton);
                    updateBlobListStates(listContainer);
                });
                listContainer.appendChild(addButton);
                div.appendChild(listContainer);
                updateBlobListStates(listContainer);
            } else if (pathString === 'CylonLED.Step') {
                const container = document.createElement('div');
                container.className = 'number-input-container';
                const input = createNumberInput(value, 0.1, undefined, 0.1);
                input.id = pathString;
                input.dataset.path = pathString;
                container.appendChild(input);
                div.appendChild(container);
            } else if (pathString === 'CylonLED.Width') {
                const container = document.createElement('div');
                container.className = 'number-input-container';
                const input = createNumberInput(value, 1, ledsTotal > 0 ? Math.floor(ledsTotal / 2) : 1, 1);
                input.id = pathString;
                input.dataset.path = pathString;
                container.appendChild(input);
                div.appendChild(container);
            } else if (pathString === 'NightLED.Latitude') {
                const container = document.createElement('div');
                container.className = 'number-input-container';
                const input = createNumberInput(value, -90, 90, 0.000001);
                input.id = pathString;
                input.dataset.path = pathString;
                container.appendChild(input);
                div.appendChild(container);
            } else if (pathString === 'NightLED.Longitude') {
                const container = document.createElement('div');
                container.className = 'number-input-container';
                const input = createNumberInput(value, -180, 180, 0.000001);
                input.id = pathString;
                input.dataset.path = pathString;
                container.appendChild(input);
                div.appendChild(container);
            } else if (isNonNegativeNumber(key)) {
                const container = document.createElement('div');
                container.className = 'number-input-container';
                const input = createNumberInput(value, 0, undefined, 1);
                input.id = pathString;
                input.dataset.path = pathString;
                container.appendChild(input);
                div.appendChild(container);
            } else if (isDbField(key)) {
                const container = document.createElement('div');
                container.className = 'number-input-container';
                const input = createNumberInput(value, undefined, 0, 1);
                input.id = pathString;
                input.dataset.path = pathString;
                container.appendChild(input);
                div.appendChild(container);
            } else if (isLedPosition(key)) {
                const container = document.createElement('div');
                container.className = 'number-input-container';
                const input = createNumberInput(value, 0, ledsTotal > 0 ? ledsTotal - 1 : 0, 1);
                input.id = pathString;
                input.dataset.path = pathString;
                container.appendChild(input);
                div.appendChild(container);
            } else if (isDurationField(key)) {
                const container = document.createElement('div');
                container.className = 'duration-input-container';
                const input = createNumberInput(value / 1000000, 0, undefined, 1);
                input.id = pathString;
                input.dataset.type = 'duration';
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
                container.appendChild(createLabeledInput('R', r, 0, 255, 1));
                container.appendChild(createLabeledInput('G', g, 0, 255, 1));
                container.appendChild(createLabeledInput('B', b, 0, 255, 1));
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

function updateArrowStates(listContainer) {
    const items = listContainer.querySelectorAll('.color-list-item');
    const numItems = items.length;
    items.forEach((item, index) => {
        const upArrow = item.querySelector('.arrow-up');
        const downArrow = item.querySelector('.arrow-down');
        const deleteBtn = item.querySelector('.delete-color-btn');

        if (upArrow) upArrow.classList.toggle('disabled', index === 0);
        if (downArrow) downArrow.classList.toggle('disabled', index === numItems - 1);
        if (deleteBtn) deleteBtn.classList.toggle('disabled', numItems === 1);
    });
}

function updateBlobListStates(listContainer) {
    const items = listContainer.querySelectorAll('.blob-list-item');
    const numItems = items.length;
    items.forEach((item) => {
        const deleteBtn = item.querySelector('.delete-blob-btn');
        if (deleteBtn) {
            deleteBtn.classList.toggle('disabled', numItems === 1);
        }
    });
}


function createColorListItem(color) {
    const item = document.createElement('div');
    item.className = 'color-list-item';

    const moveHandle = document.createElement('div');
    moveHandle.className = 'move-handle';
    const upArrow = document.createElement('span');
    upArrow.className = 'arrow-up';
    upArrow.innerHTML = '&#9650;'; // Up-pointing triangle
    upArrow.addEventListener('click', () => {
        if (item.previousElementSibling) {
            item.parentElement.insertBefore(item, item.previousElementSibling);
            updateArrowStates(item.parentElement);
        }
    });

    const downArrow = document.createElement('span');
    downArrow.className = 'arrow-down';
    downArrow.innerHTML = '&#9660;'; // Down-pointing triangle
    downArrow.addEventListener('click', () => {
        if (item.nextElementSibling && item.nextElementSibling.tagName === 'DIV') {
            item.parentElement.insertBefore(item.nextElementSibling, item);
            updateArrowStates(item.parentElement);
        }
    });

    moveHandle.appendChild(upArrow);
    moveHandle.appendChild(downArrow);
    item.appendChild(moveHandle);


    const [r, g, b] = color;
    item.appendChild(createLabeledInput('R', r, 0, 255, 1));
    item.appendChild(createLabeledInput('G', g, 0, 255, 1));
    item.appendChild(createLabeledInput('B', b, 0, 255, 1));

    const deleteBtn = document.createElement('span');
    deleteBtn.className = 'delete-color-btn';
    deleteBtn.textContent = '✖';
    deleteBtn.addEventListener('click', () => {
        const listContainer = item.parentElement;
        if (listContainer.querySelectorAll('.color-list-item').length > 1) {
            item.remove();
            updateArrowStates(listContainer);
        }
    });
    item.appendChild(deleteBtn);

    return item;
}

function createBlobListItem(blob, ledsTotal) {
    const item = document.createElement('div');
    item.className = 'blob-list-item';

    const controlsContainer = document.createElement('div');
    controlsContainer.className = 'blob-item-controls';

    const topRow = document.createElement('div');
    topRow.className = 'blob-list-item-row';
    topRow.appendChild(createLabeledInput('DeltaX', blob.DeltaX, undefined, undefined, 0.1));
    topRow.appendChild(createLabeledInput('X', blob.X, 0, ledsTotal > 0 ? ledsTotal - 1 : 0, 1));
    topRow.appendChild(createLabeledInput('Width', blob.Width, 0, undefined, 1));
    controlsContainer.appendChild(topRow);

    const bottomRow = document.createElement('div');
    bottomRow.className = 'blob-list-item-row';
    const [r, g, b] = blob.LedRGB;
    const rgbContainer = document.createElement('div');
    rgbContainer.className = 'rgb-input-container';
    rgbContainer.appendChild(createLabeledInput('R', r, 0, 255, 1));
    rgbContainer.appendChild(createLabeledInput('G', g, 0, 255, 1));
    rgbContainer.appendChild(createLabeledInput('B', b, 0, 255, 1));
    bottomRow.appendChild(rgbContainer);
    controlsContainer.appendChild(bottomRow);

    item.appendChild(controlsContainer);

    const deleteBtn = document.createElement('span');
    deleteBtn.className = 'delete-blob-btn';
    deleteBtn.textContent = '✖';
    deleteBtn.addEventListener('click', () => {
        const listContainer = item.parentElement;
        if (listContainer.querySelectorAll('.blob-list-item').length > 1) {
            item.remove();
            updateBlobListStates(listContainer);
        }
    });
    item.appendChild(deleteBtn);

    return item;
}

function createNumberInput(value, min, max, step) {
    const input = document.createElement('input');
    input.type = 'number';
    if (min !== undefined) input.min = min;
    if (max !== undefined) input.max = max;
    if (step !== undefined) input.step = step;
    input.value = value;

    input.addEventListener('input', () => {
        const numValue = parseFloat(input.value);
        const minValue = input.min !== '' ? parseFloat(input.min) : -Infinity;
        const maxValue = input.max !== '' ? parseFloat(input.max) : Infinity;

        if (numValue < minValue) {
            input.value = input.min;
        }
        if (numValue > maxValue) {
            input.value = input.max;
        }
    });

    return input;
}

function createLabeledInput(labelText, value, min, max, step) {
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
    const inputs = container.querySelectorAll('input[data-path], textarea[data-path]');
    inputs.forEach(input => {
        const path = input.dataset.path;
        if (!path) return;

        const pathParts = path.split('.');
        const lastPart = pathParts[pathParts.length - 1];
        let value;

        if (input.tagName === 'TEXTAREA') {
            try {
                value = JSON.parse(input.value);
            } catch (e) {
                value = [];
            }
        } else if (input.type === 'checkbox') {
            value = input.checked;
        } else if (input.type === 'number') {
            if (input.dataset.type === 'duration') {
                value = parseFloat(input.value) * 1000000; // convert ms to ns
            } else {
                value = parseFloat(input.value);
            }
        } else {
            value = input.value;
            if (input.dataset.type === 'array') {
                try {
                    value = JSON.parse(value);
                } catch (e) {}
            }
        }

        let current = config;
        for (let i = 0; i < pathParts.length - 1; i++) {
            current = current[pathParts[i]];
        }
        if (isLedPosition(lastPart)) {
            current[lastPart] = parseInt(value, 10);
        } else {
            current[lastPart] = value;
        }
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
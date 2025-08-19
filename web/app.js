document.addEventListener('DOMContentLoaded', () => {
    const formContainer = document.getElementById('config-form');
    const saveButton = document.getElementById('save-button');
    const messageDiv = document.getElementById('message');
    let originalConfig = null; // Variable to hold the full, original config

    // Fetch config and build the form
    fetch('/api/config')
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        })
        .then(config => {
            originalConfig = config; // Store the original config
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

        // Update the originalConfig object with values from the form
        const updatedConfig = updateConfigFromForm(originalConfig, formContainer);
        
        fetch('/api/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
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
        })
        .catch(error => {
            messageDiv.textContent = `Error saving configuration: ${error}`;
            messageDiv.style.color = 'red';
        });
    });
});

// Kicks off the recursive form building process.
function buildForm(config, container) {
    container.innerHTML = ''; // Clear previous form
    buildRecursive(config, container, []);
}

// Recursively builds form elements for a given object and path.
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
            buildRecursive(value, fieldset, currentPath); // Recurse into the nested object
        } else {
            // Handle primitive values and arrays (as text for now)
            const div = document.createElement('div');
            const label = document.createElement('label');
            label.textContent = key;
            label.htmlFor = pathString;
            
            const input = document.createElement('input');
            input.id = pathString;
            input.name = pathString;
            input.dataset.path = pathString;
            input.dataset.type = Array.isArray(value) ? 'array' : typeof value;

            if (typeof value === 'boolean') {
                input.type = 'checkbox';
                input.checked = value;
            } else {
                input.type = 'text';
                input.value = Array.isArray(value) ? JSON.stringify(value) : value;
            }

            div.appendChild(label);
            div.appendChild(input);
            parentElement.appendChild(div);
        }
    }
}

// Updates the provided config object with values from the form inputs using their data-path.
function updateConfigFromForm(config, container) {
    const inputs = container.querySelectorAll('input');
    
    inputs.forEach(input => {
        const path = input.dataset.path.split('.');
        const originalType = input.dataset.type;
        let value;

        if (input.type === 'checkbox') {
            value = input.checked;
        } else {
            value = input.value;
            // Attempt to cast back to the original type
            if (originalType === 'number') {
                value = parseFloat(value);
            } else if (originalType === 'array') {
                try {
                    value = JSON.parse(value);
                } catch (e) {
                    console.error(`Invalid JSON for array field ${input.dataset.path}:`, value);
                    // Keep it as a string to show the user their error
                }
            }
        }
        
        // Traverse the config object to set the value at the correct path
        let current = config;
        for (let i = 0; i < path.length - 1; i++) {
            current = current[path[i]];
        }
        current[path[path.length - 1]] = value;
    });

    return config;
}

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

function buildForm(config, container) {
    container.innerHTML = ''; // Clear previous form
    for (const producerName in config) {
        const producerConfig = config[producerName];
        const fieldset = document.createElement('fieldset');
        const legend = document.createElement('legend');
        legend.textContent = producerName;
        fieldset.appendChild(legend);

        for (const key in producerConfig) {
            const value = producerConfig[key];
            // For now, we'll just handle simple key-value pairs.
            // We can add more complex handling for nested objects or arrays later.
            if (typeof value !== 'object' && value !== null) {
                const div = document.createElement('div');
                const label = document.createElement('label');
                label.textContent = key;
                label.htmlFor = `${producerName}-${key}`;
                
                const input = document.createElement('input');
                input.type = typeof value === 'boolean' ? 'checkbox' : 'text';
                input.id = `${producerName}-${key}`;
                input.name = `${producerName}-${key}`;
                input.dataset.producer = producerName;
                input.dataset.key = key;
                // Store original type for smarter casting on save
                input.dataset.type = typeof value;

                if (typeof value === 'boolean') {
                    input.checked = value;
                } else {
                    input.value = value;
                }

                div.appendChild(label);
                div.appendChild(input);
                fieldset.appendChild(div);
            }
        }
        container.appendChild(fieldset);
    }
}

// Updates the provided config object with values from the form inputs.
function updateConfigFromForm(config, container) {
    const inputs = container.querySelectorAll('input');
    
    inputs.forEach(input => {
        const producer = input.dataset.producer;
        const key = input.dataset.key;
        const originalType = input.dataset.type;
        let value;

        if (input.type === 'checkbox') {
            value = input.checked;
        } else {
            value = input.value;
            // Attempt to cast back to the original type
            if (originalType === 'number') {
                value = parseFloat(value);
            }
        }
        
        // Update the value in the original config object
        if (config[producer] && typeof config[producer][key] !== 'undefined') {
            config[producer][key] = value;
        }
    });

    return config;
}

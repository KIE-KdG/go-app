(function() {
  const chatForm = document.getElementById('chat-form');
  const messageInput = document.getElementById('message-input');
  // This is the element where all messages will be appended.
  const messages = document.getElementById('messages');

  // Handle the form submission.
  chatForm.addEventListener('submit', function(event) {
    event.preventDefault();

    const message = messageInput.value.trim();
    if (!message) return;

    // Append the user's message.
    appendMessage('user', message);

    // Append a loading bubble that just says "thinking.."
    const loadingElem = appendLoadingMessage('bot');

    // Send the prompt to the backend.
    fetch('/api/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: message })
    })
    .then(response => {
      if (!response.ok) throw new Error("Network response was not ok");
      return response.json();
    })
    .then(data => {
      // Remove the loading element.
      loadingElem.remove();
      // Append the bot's text response.
      appendMessage('bot', data.response);

      // Now fetch the GeoJSON data.
      return fetch('/api/geojson', {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' }
      });
    })
    .then(response => {
      if (!response.ok) throw new Error("GeoJSON network response was not ok");
      return response.json();
    })
    .then(geojsonData => {
      // Append the map bubble with the GeoJSON data.
      appendMap('bot', geojsonData);
    })
    .catch(err => {
      console.error('Error:', err);
      // Remove loading message if still present.
      const existingLoading = document.querySelector('.messages.bot.loading');
      if (existingLoading) {
        existingLoading.remove();
      }
      appendMessage('bot', 'Sorry, there was an error processing your request.');
    });

    // Clear the input field.
    messageInput.value = '';
  });

  // Helper function to append text messages as chat bubbles.
  function appendMessage(sender, text) {
    // Create a wrapper div using Bootstrap flex utilities for alignment.
    const wrapper = document.createElement('div');
    wrapper.classList.add('d-flex', 'mb-2');
    if (sender === 'user') {
      wrapper.classList.add('justify-content-end');
    } else {
      wrapper.classList.add('justify-content-start');
    }
    
    // Create the bubble element.
    const bubble = document.createElement('div');
    bubble.classList.add('p-2', 'rounded');
    bubble.style.maxWidth = '70%';
    bubble.textContent = text;
    
    // Apply different background classes based on the sender.
    if (sender === 'user') {
      bubble.classList.add('bg-primary', 'text-white');
    } else {
      bubble.classList.add('bg-light', 'text-dark');
    }
    
    // Append the bubble to the wrapper.
    wrapper.appendChild(bubble);
    // Append the wrapper to the messages element.
    messages.appendChild(wrapper);
    // Scroll the messages container to the bottom.
    messages.scrollTop = messages.scrollHeight;
  }

  // Helper function to append a loading message that displays "thinking.."
  function appendLoadingMessage(sender) {
    const loadingElem = document.createElement('div');
    // Using the same styling classes can help you later style these messages differently if needed.
    loadingElem.classList.add('messages', sender, 'loading');
    loadingElem.textContent = "thinking..";
    messages.appendChild(loadingElem);
    messages.scrollTop = messages.scrollHeight;
    return loadingElem;
  }

  // Helper function to append a Leaflet map displaying the GeoJSON.
  function appendMap(sender, geojsonData) {
    const container = document.createElement('div');
    container.classList.add('messages', sender);
    container.style.width = '300px';
    container.style.height = '300px';
    container.style.marginTop = '10px';
    container.style.border = '1px solid #ccc';
    container.style.borderRadius = '5px';
    container.style.overflow = 'hidden';
    
    const mapDiv = document.createElement('div');
    mapDiv.style.width = '100%';
    mapDiv.style.height = '100%';
    container.appendChild(mapDiv);
    
    messages.appendChild(container);
    messages.scrollTop = messages.scrollHeight;
    
    const map = L.map(mapDiv, { attributionControl: false, zoomControl: false })
      .setView([51.505, -0.09], 13);
    
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '© OpenStreetMap contributors'
    }).addTo(map);
    
    L.geoJSON(geojsonData, {
      style: function() {
        return {
          color: "#3388ff",
          weight: 2,
          opacity: 0.3,
          fillOpacity: 0.3
        };
      },
      onEachFeature: function(feature, layer) {
        if (feature.properties && feature.properties.name) {
          layer.bindPopup(feature.properties.name);
        }
      }
    }).addTo(map);
  }
})();

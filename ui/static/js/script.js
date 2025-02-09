(function() {
  const chatContainer = document.getElementById('chat-container');
  const chatForm = document.getElementById('chat-form');
  const messageInput = document.getElementById('message-input');

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
      if (document.querySelector('.message.bot.loading')) {
        document.querySelector('.message.bot.loading').remove();
      }
      appendMessage('bot', 'Sorry, there was an error processing your request.');
    });

    // Clear the input field.
    messageInput.value = '';
  });

  // Helper function to append text messages.
  function appendMessage(sender, text) {
    const messageElem = document.createElement('div');
    messageElem.classList.add('message', sender);
    messageElem.textContent = text;
    chatContainer.appendChild(messageElem);
    chatContainer.scrollTop = chatContainer.scrollHeight;
  }

  // Helper function to append a loading message that displays "thinking.."
  function appendLoadingMessage(sender) {
    const loadingElem = document.createElement('div');
    loadingElem.classList.add('message', sender, 'loading');
    loadingElem.textContent = "thinking..";
    chatContainer.appendChild(loadingElem);
    chatContainer.scrollTop = chatContainer.scrollHeight;
    return loadingElem;
  }

  // Helper function to append a Leaflet map displaying the GeoJSON.
  function appendMap(sender, geojsonData) {
    const container = document.createElement('div');
    container.classList.add('message', sender);
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
    
    chatContainer.appendChild(container);
    chatContainer.scrollTop = chatContainer.scrollHeight;
    
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

(function() {
  // Wait until the DOM is fully loaded.
  document.addEventListener('DOMContentLoaded', function() {
    // Fetch the GeoJSON data from your map handler endpoint.
    fetch('/api/geojson', {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' }
    })
    .then(function(response) {
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }
      return response.json();
    })
    .then(function(geojsonData) {
      // Initialize the Leaflet map inside the div with id "map".
      const map = L.map('map').setView([50.8999, 4.553], 8);
      
      // Add an OpenStreetMap tile layer.
      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors',
        maxZoom: 19
      }).addTo(map);
      
      // Add the GeoJSON layer to the map.
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
          // Bind a popup if the feature has a name property.
          if (feature.properties && feature.properties.name) {
            layer.bindPopup(feature.properties.name);
          }
        }
      }).addTo(map);
    })
    .catch(function(error) {
      console.error('Error fetching or processing GeoJSON data:', error);
    });
  });
})();

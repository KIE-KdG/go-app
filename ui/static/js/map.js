/**
 * map.js - GeoJSON Map Tester Functions
 * 
 * This file contains functions for initializing and manipulating 
 * maps with GeoJSON data for testing purposes.
 */

// Store map instance globally for console debugging
let mapInstance;
let geoJsonLayer;
let selectedFeature = null;

// Map initialization function
function initTestMap(containerId, geoJsonData, options = {}) {
  const defaultOptions = {
    center: [0, 0],
    zoom: 2,
    minZoom: 1
  };
  
  const mapOptions = {...defaultOptions, ...options};
  
  // Initialize map
  const map = L.map(containerId, mapOptions);
  mapInstance = map; // Store for global access
  
  // Available basemaps
  const basemaps = {
    osm: L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }),
    satellite: L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}', {
      attribution: 'Tiles &copy; Esri &mdash; Source: Esri, i-cubed, USDA, USGS, AEX, GeoEye, Getmapping, Aerogrid, IGN, IGP, UPR-EGP, and the GIS User Community'
    }),
    topo: L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
      attribution: 'Map data &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors, Imagery © <a href="https://opentopomap.org">OpenTopoMap</a>'
    })
  };
  
  // Add default basemap
  basemaps.osm.addTo(map);
  
  // Parse the GeoJSON if it's a string
  let geoJson = geoJsonData;
  if (typeof geoJsonData === 'string') {
    try {
      geoJson = JSON.parse(geoJsonData);
    } catch (e) {
      console.error('Error parsing GeoJSON:', e);
      return { map, error: 'Error parsing GeoJSON' };
    }
  }
  
  // Count features and update UI if elements exist
  let featureCount = 0;
  if (geoJson.features) {
    featureCount = geoJson.features.length;
  } else if (geoJson.type === 'Feature') {
    featureCount = 1;
  }
  
  const featureCountEl = document.getElementById('feature-count');
  if (featureCountEl) {
    featureCountEl.textContent = `${featureCount} features`;
  }
  
  // Add GeoJSON layer with styling and events
  geoJsonLayer = L.geoJSON(geoJson, {
    style: styleFeature,
    onEachFeature: bindFeatureEvents,
    pointToLayer: createMarkerForPoint
  }).addTo(map);
  
  // Try to fit map to GeoJSON bounds
  try {
    if (geoJsonLayer.getBounds().isValid()) {
      map.fitBounds(geoJsonLayer.getBounds());
    }
  } catch (e) {
    console.warn('Could not fit to bounds:', e);
  }
  
  // Set up UI controls if they exist
  setupMapControls(map, basemaps, geoJsonLayer);
  
  // Set up map event listeners
  setupMapEvents(map);
  
  return { map, geoJsonLayer, featureCount };
}

// Feature styling function
function styleFeature(feature) {
  // Check if this is the selected feature
  if (selectedFeature && feature === selectedFeature) {
    return {
      weight: 4,
      color: '#FF4500',
      dashArray: '',
      fillOpacity: 0.7,
      fillColor: '#FF8C00'
    };
  }
  
  // Default styling
  return {
    weight: 2,
    opacity: 1,
    color: '#3388ff',
    dashArray: '3',
    fillOpacity: 0.5,
    fillColor: getColorForFeature(feature)
  };
}

// Get color based on feature properties
function getColorForFeature(feature) {
  // This is a simple example - customize as needed
  if (!feature.properties) return '#3388ff';
  
  // If it has a type property, use different colors
  if (feature.properties.type) {
    const type = feature.properties.type.toLowerCase();
    if (type.includes('water') || type.includes('river') || type.includes('lake')) {
      return '#0077be';
    } else if (type.includes('forest') || type.includes('park') || type.includes('green')) {
      return '#228B22';
    } else if (type.includes('building') || type.includes('urban')) {
      return '#CD5C5C';
    }
  }
  
  return '#3388ff'; // Default blue
}

// Create markers for point features
function createMarkerForPoint(feature, latlng) {
  return L.circleMarker(latlng, {
    radius: 8,
    fillColor: getColorForFeature(feature),
    color: "#000",
    weight: 1,
    opacity: 1,
    fillOpacity: 0.8
  });
}

// Bind events to features
function bindFeatureEvents(feature, layer) {
  layer.on({
    mouseover: highlightFeature,
    mouseout: resetHighlight,
    click: clickFeature
  });
}

// Highlight feature on mouseover
function highlightFeature(e) {
  const layer = e.target;
  
  if (layer !== selectedFeature) {
    layer.setStyle({
      weight: 3,
      color: '#666',
      dashArray: '',
      fillOpacity: 0.7
    });
  
    if (!L.Browser.ie && !L.Browser.opera && !L.Browser.edge) {
      layer.bringToFront();
    }
  }
}

// Reset highlight on mouseout
function resetHighlight(e) {
  if (e.target !== selectedFeature) {
    geoJsonLayer.resetStyle(e.target);
  }
}

// Handle feature click
function clickFeature(e) {
  const layer = e.target;
  const feature = layer.feature;
  
  // Reset previously selected feature
  if (selectedFeature) {
    geoJsonLayer.resetStyle(selectedFeature);
  }
  
  // Set new selected feature
  selectedFeature = feature;
  
  // Apply selected style
  layer.setStyle({
    weight: 4,
    color: '#FF4500',
    dashArray: '',
    fillOpacity: 0.7,
    fillColor: '#FF8C00'
  });
  
  // Show feature properties
  showFeatureProperties(feature);
  
  // Zoom to feature
  if (layer.getBounds) {
    mapInstance.fitBounds(layer.getBounds());
  } else if (layer.getLatLng) {
    mapInstance.setView(layer.getLatLng(), mapInstance.getZoom() < 12 ? 12 : mapInstance.getZoom());
  }
}

// Show feature properties in info panel
function showFeatureProperties(feature) {
  const featureInfo = document.getElementById('feature-info');
  if (!featureInfo) return;
  
  if (feature && feature.properties) {
    featureInfo.textContent = JSON.stringify(feature.properties, null, 2);
  } else {
    featureInfo.textContent = 'No properties found for this feature.';
  }
}

// Set up map controls (if they exist in the DOM)
function setupMapControls(map, basemaps, geoJsonLayer) {
  // Basemap selector
  const basemapSelector = document.getElementById('basemap-selector');
  if (basemapSelector) {
    basemapSelector.addEventListener('change', function(e) {
      // Remove all basemaps
      Object.values(basemaps).forEach(layer => {
        if (map.hasLayer(layer)) {
          map.removeLayer(layer);
        }
      });
      
      // Add selected basemap
      basemaps[e.target.value].addTo(map);
    });
  }
  
  // Reset view button
  const resetViewBtn = document.getElementById('reset-view-btn');
  if (resetViewBtn) {
    resetViewBtn.addEventListener('click', function() {
      try {
        if (geoJsonLayer.getBounds().isValid()) {
          map.fitBounds(geoJsonLayer.getBounds());
        } else {
          map.setView([0, 0], 2);
        }
      } catch (e) {
        map.setView([0, 0], 2);
      }
      
      // Clear selected feature
      if (selectedFeature) {
        geoJsonLayer.resetStyle(selectedFeature);
        selectedFeature = null;
      }
      
      // Clear feature info
      const featureInfo = document.getElementById('feature-info');
      if (featureInfo) {
        featureInfo.textContent = 'No feature selected';
      }
    });
  }
  
  // Toggle GeoJSON button
  const toggleGeoJsonBtn = document.getElementById('toggle-geojson-btn');
  if (toggleGeoJsonBtn) {
    let geoJsonVisible = true;
    toggleGeoJsonBtn.addEventListener('click', function() {
      if (geoJsonVisible) {
        map.removeLayer(geoJsonLayer);
        this.classList.remove('btn-secondary');
        this.classList.add('btn-outline');
        this.textContent = 'Show GeoJSON Layer';
      } else {
        map.addLayer(geoJsonLayer);
        this.classList.remove('btn-outline');
        this.classList.add('btn-secondary');
        this.textContent = 'Hide GeoJSON Layer';
      }
      geoJsonVisible = !geoJsonVisible;
    });
  }
}

// Set up map event listeners
function setupMapEvents(map) {
  // Update zoom level display
  const zoomLevelEl = document.getElementById('zoom-level');
  if (zoomLevelEl) {
    map.on('zoomend', function() {
      zoomLevelEl.textContent = `Zoom: ${map.getZoom()}`;
    });
  }
  
  // Update map coordinates on mouse move
  const mapCoords = document.getElementById('map-coords');
  if (mapCoords) {
    map.on('mousemove', function(e) {
      mapCoords.textContent = `Lat: ${e.latlng.lat.toFixed(5)}, Lng: ${e.latlng.lng.toFixed(5)}`;
    });
  }
}

// Initialize map when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
  const mapContainer = document.getElementById('map-container');
  const geoJsonPreview = document.getElementById('geojson-preview');
  
  if (mapContainer && geoJsonPreview) {
    try {
      const { map, error } = initTestMap('map-container', geoJsonPreview.textContent);
      
      if (error) {
        const mapStatus = document.getElementById('map-status');
        if (mapStatus) {
          mapStatus.textContent = error;
          mapStatus.classList.add('text-error');
        }
      } else {
        const mapStatus = document.getElementById('map-status');
        if (mapStatus) {
          mapStatus.textContent = 'Map initialized successfully';
        }
      }
    } catch (e) {
      console.error('Error initializing map:', e);
    }
  }
});
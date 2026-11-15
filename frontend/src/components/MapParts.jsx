import { useEffect } from "react";
import { MapContainer, Polygon, Popup, TileLayer, useMap, useMapEvents } from "react-leaflet";
import { DEFAULT_CENTER, DEFAULT_ZOOM, categoryColor, categoryLabel } from "../constants";

export function BaseMap({ children }) {
  return (
    <MapContainer center={DEFAULT_CENTER} zoom={DEFAULT_ZOOM} className="map" scrollWheelZoom>
      <TileLayer
        attribution="&copy; OpenStreetMap contributors"
        url="https://tile.openstreetmap.org/{z}/{x}/{y}.png"
      />
      {children}
    </MapContainer>
  );
}

// Draws every geofence as a coloured polygon.
// interactive=false lets clicks pass through to the map (needed for drawing and picking points).
export function GeofenceLayer({ geofences, interactive = true }) {
  return (
    <>
      {geofences.map((g) => (
        <Polygon
          key={g.id}
          positions={g.coordinates}
          pathOptions={{
            color: categoryColor(g.category),
            weight: 2,
            fillOpacity: 0.2,
            interactive,
          }}
        >
          {interactive && (
            <Popup>
              <strong>{g.name}</strong>
              <br />
              {categoryLabel(g.category)}
              {g.description ? (
                <>
                  <br />
                  {g.description}
                </>
              ) : null}
            </Popup>
          )}
        </Polygon>
      ))}
    </>
  );
}

// Zooms the map so all geofences are visible.
export function FitToGeofences({ geofences }) {
  const map = useMap();
  useEffect(() => {
    const points = geofences.flatMap((g) => g.coordinates);
    if (points.length > 0) {
      map.fitBounds(points, { padding: [30, 30], maxZoom: 15 });
    }
  }, [geofences, map]);
  return null;
}

// Reports the latitude and longitude of every click on the map.
export function MapClick({ onClick }) {
  useMapEvents({
    click(event) {
      onClick(event.latlng.lat, event.latlng.lng);
    },
  });
  return null;
}
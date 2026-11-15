import { useEffect, useState } from "react";
import { CircleMarker } from "react-leaflet";
import toast from "react-hot-toast";
import { api } from "../api";
import { categoryLabel } from "../constants";
import { BaseMap, FitToGeofences, GeofenceLayer, MapClick } from "../components/MapParts";
import { toInputValue } from "../utils";

export default function UpdateLocation() {
  const [vehicles, setVehicles] = useState([]);
  const [geofences, setGeofences] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [form, setForm] = useState({
    vehicle_id: "",
    latitude: "",
    longitude: "",
    timestamp: toInputValue(new Date()),
  });
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState(null);

  useEffect(() => {
    async function load() {
      try {
        const [vehicleData, geofenceData] = await Promise.all([api("/vehicles"), api("/geofences")]);
        setVehicles(vehicleData.vehicles || []);
        setGeofences(geofenceData.geofences || []);
      } catch (error) {
        setLoadError(error.message);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  function updateField(event) {
    setForm({ ...form, [event.target.name]: event.target.value });
  }

  function pickOnMap(lat, lng) {
    setForm((previous) => ({ ...previous, latitude: lat.toFixed(6), longitude: lng.toFixed(6) }));
  }

  const lat = parseFloat(form.latitude);
  const lng = parseFloat(form.longitude);
  const hasPoint = !Number.isNaN(lat) && !Number.isNaN(lng);

  async function handleSubmit(event) {
    event.preventDefault();

    if (!form.vehicle_id) return toast.error("Please select a vehicle");
    if (!hasPoint) return toast.error("Click on the map or enter a latitude and longitude");
    if (lat < -90 || lat > 90) return toast.error("Latitude must be between -90 and 90");
    if (lng < -180 || lng > 180) return toast.error("Longitude must be between -180 and 180");
    const when = new Date(form.timestamp);
    if (Number.isNaN(when.getTime())) return toast.error("Please choose a valid date and time");

    setSubmitting(true);
    try {
      const data = await api("/vehicles/location", {
        method: "POST",
        body: JSON.stringify({
          vehicle_id: form.vehicle_id,
          latitude: lat,
          longitude: lng,
          timestamp: when.toISOString(),
        }),
      });
      setResult({ ...data, vehicle: vehicles.find((v) => v.id === form.vehicle_id) });
      toast.success("Location updated");
    } catch (error) {
      toast.error(error.message);
    } finally {
      setSubmitting(false);
    }
  }

  const inside = result?.current_geofences || [];

  return (
    <div className="two-column">
      <section className="card">
        <h2>Update vehicle location</h2>
        {loadError && <p className="error">Could not load data: {loadError}</p>}
        {!loading && !loadError && vehicles.length === 0 && (
          <p className="error">Register a vehicle first on the Vehicles page.</p>
        )}

        <form onSubmit={handleSubmit} className="form">
          <label>
            Vehicle
            <select name="vehicle_id" value={form.vehicle_id} onChange={updateField} required>
              <option value="">{loading ? "Loading vehicles..." : "Select a vehicle"}</option>
              {vehicles.map((v) => (
                <option key={v.id} value={v.id}>{v.vehicle_number} - {v.driver_name}</option>
              ))}
            </select>
          </label>

          <div className="form-row">
            <label>
              Latitude
              <input name="latitude" type="number" step="any" value={form.latitude}
                onChange={updateField} placeholder="37.7849" />
            </label>
            <label>
              Longitude
              <input name="longitude" type="number" step="any" value={form.longitude}
                onChange={updateField} placeholder="-122.4194" />
            </label>
          </div>
          <p className="hint">Tip: click anywhere on the map to fill the latitude and longitude.</p>

          <label>
            Date and time
            <input name="timestamp" type="datetime-local" value={form.timestamp}
              onChange={updateField} required />
          </label>
          <button type="button" className="secondary"
            onClick={() => setForm({ ...form, timestamp: toInputValue(new Date()) })}>
            Use current time
          </button>

          <button type="submit" disabled={submitting || loading}>
            {submitting ? "Sending..." : "Send location update"}
          </button>
        </form>

        {result && (
          <div className="result">
            <h3>Result for {result.vehicle?.vehicle_number || result.vehicle_id}</h3>
            {inside.length === 0 ? (
              <p>The vehicle is <strong>outside</strong> all geofences.</p>
            ) : (
              <>
                <p>The vehicle is <strong>inside</strong> {inside.length} geofence{inside.length > 1 ? "s" : ""}:</p>
                <ul>
                  {inside.map((g) => (
                    <li key={g.geofence_id}>{g.geofence_name}</li>
                  ))}
                </ul>
              </>
            )}
            <p className="muted">
              If an alert rule matches an entry or exit, the alert appears in Live Alerts.
            </p>
          </div>
        )}
      </section>

      <section className="card">
        <h2>Pick the position on the map</h2>
        <div className="legend">
          <span>Click the map to move the vehicle marker</span>
        </div>
        <BaseMap>
          <FitToGeofences geofences={geofences} />
          <GeofenceLayer geofences={geofences} interactive={false} />
          <MapClick onClick={pickOnMap} />
          {hasPoint && (
            <CircleMarker center={[lat, lng]} radius={9}
              pathOptions={{ color: "#101828", fillColor: "#f79009", fillOpacity: 1, interactive: false }} />
          )}
        </BaseMap>
        {geofences.length > 0 && (
          <p className="muted">
            Categories shown: {[...new Set(geofences.map((g) => categoryLabel(g.category)))].join(", ")}
          </p>
        )}
      </section>
    </div>
  );
}
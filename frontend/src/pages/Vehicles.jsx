import { useCallback, useEffect, useState } from "react";
import { CircleMarker, Popup } from "react-leaflet";
import toast from "react-hot-toast";
import { api } from "../api";
import { useAlerts } from "../alerts-context";
import { categoryColor, categoryLabel } from "../constants";
import { BaseMap, FitToGeofences, GeofenceLayer } from "../components/MapParts";
import { formatCoord, formatDateTime } from "../utils";

const VEHICLE_TYPES = ["truck", "car", "van", "motorcycle"];
const EMPTY_FORM = { vehicle_number: "", driver_name: "", vehicle_type: "truck", phone: "" };

export default function Vehicles() {
  const { alerts } = useAlerts();
  const [vehicles, setVehicles] = useState([]);
  const [geofences, setGeofences] = useState([]);
  const [locations, setLocations] = useState({});
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [form, setForm] = useState(EMPTY_FORM);
  const [saving, setSaving] = useState(false);

  const loadAll = useCallback(async () => {
    try {
      const [vehicleData, geofenceData] = await Promise.all([api("/vehicles"), api("/geofences")]);
      const vehicleList = vehicleData.vehicles || [];
      setVehicles(vehicleList);
      setGeofences(geofenceData.geofences || []);

      // A vehicle with no location yet returns an error, so failures are simply skipped.
      const results = await Promise.allSettled(
        vehicleList.map((v) => api(`/vehicles/location/${v.id}`)),
      );
      const next = {};
      results.forEach((result, index) => {
        if (result.status === "fulfilled") next[vehicleList[index].id] = result.value;
      });
      setLocations(next);
      setLoadError("");
    } catch (error) {
      setLoadError(error.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAll();
    const timer = setInterval(loadAll, 10000);
    return () => clearInterval(timer);
  }, [loadAll]);

  // Refresh right away when a live alert arrives.
  const latestAlertId = alerts[0]?.event_id;
  useEffect(() => {
    if (latestAlertId) loadAll();
  }, [latestAlertId, loadAll]);

  function updateField(event) {
    setForm({ ...form, [event.target.name]: event.target.value });
  }

  async function handleSubmit(event) {
    event.preventDefault();
    setSaving(true);
    try {
      const created = await api("/vehicles", { method: "POST", body: JSON.stringify(form) });
      toast.success(`Vehicle ${created.vehicle_number} registered`);
      setForm(EMPTY_FORM);
      await loadAll();
    } catch (error) {
      toast.error(error.message);
    } finally {
      setSaving(false);
    }
  }

  const trackedVehicles = vehicles.filter((v) => locations[v.id]?.current_location);

  return (
    <div className="stack">
      <div className="two-column">
        <section className="card">
          <h2>Register a vehicle</h2>
          <form onSubmit={handleSubmit} className="form">
            <label>
              Vehicle number
              <input name="vehicle_number" value={form.vehicle_number} onChange={updateField}
                placeholder="KA-01-AB-1234" required />
            </label>
            <label>
              Driver name
              <input name="driver_name" value={form.driver_name} onChange={updateField}
                placeholder="John Doe" required />
            </label>
            <label>
              Vehicle type
              <select name="vehicle_type" value={form.vehicle_type} onChange={updateField}>
                {VEHICLE_TYPES.map((type) => (
                  <option key={type} value={type}>{type}</option>
                ))}
              </select>
            </label>
            <label>
              Phone
              <input name="phone" value={form.phone} onChange={updateField}
                placeholder="+911234567890" required />
            </label>
            <button type="submit" disabled={saving}>
              {saving ? "Saving..." : "Register vehicle"}
            </button>
          </form>
        </section>

        <section className="card">
          <div className="card-header">
            <h2>Registered vehicles</h2>
            <button type="button" className="secondary" onClick={loadAll}>Refresh</button>
          </div>
          {loading && <p className="muted">Loading vehicles...</p>}
          {loadError && <p className="error">Could not load vehicles: {loadError}</p>}
          {!loading && !loadError && vehicles.length === 0 && (
            <p className="muted">No vehicles yet. Register one using the form.</p>
          )}
          {vehicles.length > 0 && (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Number</th><th>Driver</th><th>Type</th><th>Phone</th>
                    <th>Last location</th><th>Inside geofences</th>
                  </tr>
                </thead>
                <tbody>
                  {vehicles.map((v) => {
                    const info = locations[v.id];
                    const location = info?.current_location;
                    const inside = info?.current_geofences || [];
                    return (
                      <tr key={v.id}>
                        <td>{v.vehicle_number}</td>
                        <td>{v.driver_name}</td>
                        <td>{v.vehicle_type}</td>
                        <td>{v.phone}</td>
                        <td>
                          {location ? (
                            <>
                              {formatCoord(location.latitude)}, {formatCoord(location.longitude)}
                              <div className="muted">{formatDateTime(location.timestamp)}</div>
                            </>
                          ) : (
                            <span className="muted">No location yet</span>
                          )}
                        </td>
                        <td>
                          {!location && <span className="muted">-</span>}
                          {location && inside.length === 0 && <span className="muted">Outside all</span>}
                          {inside.map((g) => (
                            <span key={g.geofence_id} className="badge badge-cat"
                              style={{ background: categoryColor(g.category) }}>
                              {g.geofence_name}
                            </span>
                          ))}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      <section className="card">
        <h2>Vehicles on the map</h2>
        {!loading && trackedVehicles.length === 0 && (
          <p className="muted">
            No vehicle has sent a location yet. Use the Update Location page to place one.
          </p>
        )}
        <BaseMap>
          <FitToGeofences geofences={geofences} />
          <GeofenceLayer geofences={geofences} />
          {trackedVehicles.map((v) => {
            const info = locations[v.id];
            const inside = info.current_geofences || [];
            const inRestricted = inside.some((g) => g.category === "restricted_zone");
            return (
              <CircleMarker
                key={v.id}
                center={[info.current_location.latitude, info.current_location.longitude]}
                radius={9}
                pathOptions={{
                  color: "#101828",
                  fillColor: inRestricted ? "#d92d20" : "#2f6fed",
                  fillOpacity: 1,
                }}
              >
                <Popup>
                  <strong>{v.vehicle_number}</strong> ({v.driver_name})
                  <br />
                  {inside.length === 0
                    ? "Outside all geofences"
                    : inside.map((g) => `${g.geofence_name} (${categoryLabel(g.category)})`).join(", ")}
                </Popup>
              </CircleMarker>
            );
          })}
        </BaseMap>
        <p className="muted">Red marker: vehicle inside a restricted zone. Blue marker: anywhere else.</p>
      </section>
    </div>
  );
}
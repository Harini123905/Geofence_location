import { useCallback, useEffect, useState } from "react";
import toast from "react-hot-toast";
import { api } from "../api";
import { categoryLabel } from "../constants";
import { formatDateTime } from "../utils";

const EVENT_TYPES = [
  { value: "entry", label: "Entry (vehicle enters the zone)" },
  { value: "exit", label: "Exit (vehicle leaves the zone)" },
  { value: "both", label: "Both entry and exit" },
];

const EMPTY_FORM = { geofence_id: "", vehicle_id: "", event_type: "both" };

export default function AlertRules() {
  const [geofences, setGeofences] = useState([]);
  const [vehicles, setVehicles] = useState([]);
  const [rules, setRules] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [form, setForm] = useState(EMPTY_FORM);
  const [saving, setSaving] = useState(false);

  const loadAll = useCallback(async () => {
    try {
      const [geofenceData, vehicleData, alertData] = await Promise.all([
        api("/geofences"),
        api("/vehicles"),
        api("/alerts"),
      ]);
      setGeofences(geofenceData.geofences || []);
      setVehicles(vehicleData.vehicles || []);
      setRules(alertData.alerts || []);
      setLoadError("");
    } catch (error) {
      setLoadError(error.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  function updateField(event) {
    setForm({ ...form, [event.target.name]: event.target.value });
  }

  async function handleSubmit(event) {
    event.preventDefault();
    if (!form.geofence_id) {
      toast.error("Please select a geofence");
      return;
    }

    // An empty vehicle means the rule applies to all vehicles.
    const body = { geofence_id: form.geofence_id, event_type: form.event_type };
    if (form.vehicle_id) body.vehicle_id = form.vehicle_id;

    setSaving(true);
    try {
      await api("/alerts/configure", { method: "POST", body: JSON.stringify(body) });
      toast.success("Alert rule created");
      setForm(EMPTY_FORM);
      await loadAll();
    } catch (error) {
      toast.error(error.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="two-column">
      <section className="card">
        <h2>Create an alert rule</h2>
        <p className="hint">A live alert is sent only when a vehicle event matches a rule.</p>

        {!loading && !loadError && geofences.length === 0 && (
          <p className="error">Create a geofence first. A rule needs a geofence to watch.</p>
        )}

        <form onSubmit={handleSubmit} className="form">
          <label>
            Geofence
            <select name="geofence_id" value={form.geofence_id} onChange={updateField} required>
              <option value="">{loading ? "Loading geofences..." : "Select a geofence"}</option>
              {geofences.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name} ({categoryLabel(g.category)})
                </option>
              ))}
            </select>
          </label>

          <label>
            Vehicle (optional)
            <select name="vehicle_id" value={form.vehicle_id} onChange={updateField}>
              <option value="">All vehicles</option>
              {vehicles.map((v) => (
                <option key={v.id} value={v.id}>
                  {v.vehicle_number} - {v.driver_name}
                </option>
              ))}
            </select>
          </label>

          <label>
            Event type
            <select name="event_type" value={form.event_type} onChange={updateField}>
              {EVENT_TYPES.map((type) => (
                <option key={type.value} value={type.value}>{type.label}</option>
              ))}
            </select>
          </label>

          <button type="submit" disabled={saving || loading || geofences.length === 0}>
            {saving ? "Saving..." : "Create rule"}
          </button>
        </form>
      </section>

      <section className="card">
        <div className="card-header">
          <h2>Configured rules</h2>
          <button type="button" className="secondary" onClick={loadAll}>Refresh</button>
        </div>
        {loading && <p className="muted">Loading rules...</p>}
        {loadError && <p className="error">Could not load data: {loadError}</p>}
        {!loading && !loadError && rules.length === 0 && (
          <p className="muted">No alert rules yet. Create one using the form.</p>
        )}
        {rules.length > 0 && (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Geofence</th><th>Vehicle</th><th>Event</th><th>Status</th><th>Created</th>
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr key={rule.alert_id}>
                    <td>{rule.geofence_name}</td>
                    <td>{rule.vehicle_number || "All vehicles"}</td>
                    <td><span className="badge">{rule.event_type}</span></td>
                    <td><span className="badge">{rule.status}</span></td>
                    <td>{formatDateTime(rule.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
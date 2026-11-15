import { useEffect, useMemo, useState } from "react";
import { api } from "../api";
import { useAlerts } from "../alerts-context";
import { formatCoord, formatDateTime } from "../utils";

const PAGE_SIZES = [10, 20, 50, 100];
const NO_FILTERS = { vehicle_id: "", geofence_id: "", start: "", end: "" };

export default function History() {
  const { alerts } = useAlerts();
  const [vehicles, setVehicles] = useState([]);
  const [geofences, setGeofences] = useState([]);
  const [filters, setFilters] = useState(NO_FILTERS);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);
  const [data, setData] = useState({ violations: [], total_count: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Dropdown options.
  useEffect(() => {
    Promise.all([api("/vehicles"), api("/geofences")])
      .then(([vehicleData, geofenceData]) => {
        setVehicles(vehicleData.vehicles || []);
        setGeofences(geofenceData.geofences || []);
      })
      .catch(() => {
        // The table still works without the dropdown options.
      });
  }, []);

  const rangeError =
    filters.start && filters.end && new Date(filters.end) < new Date(filters.start)
      ? "The end date must be after the start date."
      : "";

  const query = useMemo(() => {
    const params = new URLSearchParams();
    if (filters.vehicle_id) params.set("vehicle_id", filters.vehicle_id);
    if (filters.geofence_id) params.set("geofence_id", filters.geofence_id);
    if (filters.start) params.set("start_date", new Date(filters.start).toISOString());
    if (filters.end) params.set("end_date", new Date(filters.end).toISOString());
    params.set("limit", String(pageSize));
    params.set("offset", String(page * pageSize));
    return params.toString();
  }, [filters, page, pageSize]);

  // Reload when filters or page change, and when a new live alert arrives.
  const latestAlertId = alerts[0]?.event_id;
  useEffect(() => {
    if (rangeError) return;
    let cancelled = false;
    setLoading(true);
    api(`/violations/history?${query}`)
      .then((response) => {
        if (cancelled) return;
        setData({ violations: response.violations || [], total_count: response.total_count || 0 });
        setError("");
      })
      .catch((err) => {
        if (!cancelled) setError(err.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [query, rangeError, latestAlertId]);

  function updateFilter(event) {
    setFilters({ ...filters, [event.target.name]: event.target.value });
    setPage(0);
  }

  const total = data.total_count;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const from = total === 0 ? 0 : page * pageSize + 1;
  const to = Math.min(total, (page + 1) * pageSize);
  const hasFilters = Object.values(filters).some(Boolean);

  return (
    <section className="card">
      <div className="card-header">
        <h2>Violation history</h2>
        <button type="button" className="secondary" disabled={!hasFilters}
          onClick={() => { setFilters(NO_FILTERS); setPage(0); }}>
          Clear filters
        </button>
      </div>

      <div className="filters">
        <label>
          Vehicle
          <select name="vehicle_id" value={filters.vehicle_id} onChange={updateFilter}>
            <option value="">All vehicles</option>
            {vehicles.map((v) => (
              <option key={v.id} value={v.id}>{v.vehicle_number}</option>
            ))}
          </select>
        </label>
        <label>
          Geofence
          <select name="geofence_id" value={filters.geofence_id} onChange={updateFilter}>
            <option value="">All geofences</option>
            {geofences.map((g) => (
              <option key={g.id} value={g.id}>{g.name}</option>
            ))}
          </select>
        </label>
        <label>
          From
          <input type="datetime-local" name="start" value={filters.start} onChange={updateFilter} />
        </label>
        <label>
          To
          <input type="datetime-local" name="end" value={filters.end} onChange={updateFilter} />
        </label>
      </div>

      {rangeError && <p className="error">{rangeError}</p>}
      {error && <p className="error">Could not load history: {error}</p>}
      {loading && !rangeError && <p className="muted">Loading history...</p>}
      {!loading && !error && !rangeError && data.violations.length === 0 && (
        <p className="muted">
          {hasFilters ? "No events match these filters." : "No events yet. They appear when a vehicle enters or exits a geofence."}
        </p>
      )}

      {data.violations.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Time</th><th>Vehicle</th><th>Geofence</th><th>Event</th><th>Location</th>
              </tr>
            </thead>
            <tbody>
              {data.violations.map((item) => (
                <tr key={item.id}>
                  <td>{formatDateTime(item.timestamp)}</td>
                  <td>{item.vehicle_number}</td>
                  <td>{item.geofence_name}</td>
                  <td><span className={`badge badge-${item.event_type}`}>{item.event_type}</span></td>
                  <td>{formatCoord(item.latitude)}, {formatCoord(item.longitude)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="pagination">
        <span className="muted">Showing {from}-{to} of {total}</span>
        <div className="row">
          <label className="inline-label">
            Per page
            <select value={pageSize} onChange={(e) => { setPageSize(Number(e.target.value)); setPage(0); }}>
              {PAGE_SIZES.map((size) => (
                <option key={size} value={size}>{size}</option>
              ))}
            </select>
          </label>
          <button type="button" className="secondary" disabled={page === 0 || loading}
            onClick={() => setPage(page - 1)}>
            Previous
          </button>
          <span>Page {page + 1} of {totalPages}</span>
          <button type="button" className="secondary" disabled={page + 1 >= totalPages || loading}
            onClick={() => setPage(page + 1)}>
            Next
          </button>
        </div>
      </div>
    </section>
  );
}
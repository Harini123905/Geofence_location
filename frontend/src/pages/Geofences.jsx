import { useCallback, useEffect, useState } from "react";
import { CircleMarker, Polygon, Polyline } from "react-leaflet";
import toast from "react-hot-toast";
import { api } from "../api";
import { CATEGORIES, categoryColor, categoryLabel } from "../constants";
import { BaseMap, FitToGeofences, GeofenceLayer, MapClick } from "../components/MapParts";
import { formatDateTime } from "../utils";

const EMPTY_FORM = { name: "", description: "", category: "delivery_zone" };
const DRAFT_STYLE = { color: "#7a5af8", weight: 2, dashArray: "6", fillOpacity: 0.15, interactive: false };

// Reads coordinates typed as "lat, lng" lines, or pasted as JSON like [[37.7,-122.4],[...]].
function parseCoordinates(text) {
  const trimmed = text.trim();
  let pairs;

  if (trimmed.startsWith("[")) {
    try {
      pairs = JSON.parse(trimmed);
    } catch {
      throw new Error("The JSON could not be read. Check the brackets and commas.");
    }
  } else {
    pairs = trimmed
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean)
      .map((line) => {
        const parts = line.replace(/[()]/g, "").split(/[\s,]+/).filter(Boolean);
        if (parts.length !== 2) {
          throw new Error(`Cannot read "${line}". Write one point per line as: latitude, longitude`);
        }
        return [Number(parts[0]), Number(parts[1])];
      });
  }

  if (!Array.isArray(pairs)) throw new Error("Coordinates must be a list of points.");

  for (const pair of pairs) {
    if (!Array.isArray(pair) || pair.length !== 2 || pair.some((n) => typeof n !== "number" || Number.isNaN(n))) {
      throw new Error("Every point needs two numbers: latitude and longitude.");
    }
    if (pair[0] < -90 || pair[0] > 90) throw new Error(`Latitude ${pair[0]} must be between -90 and 90.`);
    if (pair[1] < -180 || pair[1] > 180) throw new Error(`Longitude ${pair[1]} must be between -180 and 180.`);
  }

  // Drop the closing point if it repeats the first one. It is added again when saving.
  const result = [...pairs];
  if (result.length > 1) {
    const first = result[0];
    const last = result[result.length - 1];
    if (first[0] === last[0] && first[1] === last[1]) result.pop();
  }
  return result;
}

function countUniquePoints(points) {
  return new Set(points.map((p) => p.join(","))).size;
}

export default function Geofences() {
  const [geofences, setGeofences] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("");
  const [form, setForm] = useState(EMPTY_FORM);
  const [points, setPoints] = useState([]);
  const [coordText, setCoordText] = useState("");
  const [saving, setSaving] = useState(false);

  const loadGeofences = useCallback(async () => {
    setLoading(true);
    try {
      const query = categoryFilter ? `?category=${categoryFilter}` : "";
      const data = await api(`/geofences${query}`);
      setGeofences(data.geofences || []);
      setLoadError("");
    } catch (error) {
      setLoadError(error.message);
    } finally {
      setLoading(false);
    }
  }, [categoryFilter]);

  useEffect(() => {
    loadGeofences();
  }, [loadGeofences]);

  function updateField(event) {
    setForm({ ...form, [event.target.name]: event.target.value });
  }

  function addPoint(lat, lng) {
    setPoints((previous) => [...previous, [Number(lat.toFixed(6)), Number(lng.toFixed(6))]]);
  }

  function applyCoordinates() {
    try {
      const parsed = parseCoordinates(coordText);
      if (countUniquePoints(parsed) < 3) {
        toast.error("A polygon needs at least 3 different points");
        return;
      }
      setPoints(parsed);
      toast.success(`${parsed.length} points loaded onto the map`);
    } catch (error) {
      toast.error(error.message);
    }
  }

  function clearDrawing() {
    setPoints([]);
    setCoordText("");
  }

  async function handleSubmit(event) {
    event.preventDefault();

    if (!form.name.trim()) {
      toast.error("Please enter a name");
      return;
    }
    if (countUniquePoints(points) < 3) {
      toast.error("Click at least 3 different points on the map, or paste coordinates");
      return;
    }

    // The API needs a closed polygon: the first point is repeated at the end.
    const coordinates = [...points, points[0]];

    setSaving(true);
    try {
      await api("/geofences", {
        method: "POST",
        body: JSON.stringify({
          name: form.name.trim(),
          description: form.description.trim(),
          category: form.category,
          coordinates,
        }),
      });
      toast.success(`Geofence "${form.name.trim()}" created`);
      setForm(EMPTY_FORM);
      clearDrawing();
      await loadGeofences();
    } catch (error) {
      toast.error(error.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="two-column">
      <section className="card">
        <h2>Create a geofence</h2>
        <p className="hint">
          Click on the map to place the corners of the area, or paste coordinates below.
        </p>

        <form onSubmit={handleSubmit} className="form">
          <label>
            Name
            <input name="name" value={form.name} onChange={updateField}
              placeholder="Downtown Delivery Zone" required />
          </label>
          <label>
            Description (optional)
            <input name="description" value={form.description} onChange={updateField}
              placeholder="Main delivery area" />
          </label>
          <label>
            Category
            <select name="category" value={form.category} onChange={updateField}>
              {CATEGORIES.map((c) => (
                <option key={c.value} value={c.value}>{c.label}</option>
              ))}
            </select>
          </label>

          <div className="drawing-status">
            <strong>{points.length}</strong> point{points.length === 1 ? "" : "s"} placed
            <div className="row">
              <button type="button" className="secondary"
                onClick={() => setPoints((p) => p.slice(0, -1))} disabled={points.length === 0}>
                Undo last point
              </button>
              <button type="button" className="secondary" onClick={clearDrawing}
                disabled={points.length === 0 && !coordText}>
                Clear
              </button>
            </div>
          </div>

          <label>
            Or paste coordinates (latitude, longitude per line)
            <textarea rows={5} value={coordText} onChange={(e) => setCoordText(e.target.value)}
              placeholder={"37.7749, -122.4194\n37.7849, -122.4194\n37.7849, -122.4094\n37.7749, -122.4094"} />
          </label>
          <button type="button" className="secondary" onClick={applyCoordinates}
            disabled={!coordText.trim()}>
            Show these coordinates on the map
          </button>

          <button type="submit" disabled={saving}>
            {saving ? "Saving..." : "Save geofence"}
          </button>
        </form>
      </section>

      <div className="stack">
        <section className="card">
          <div className="card-header">
            <h2>Map</h2>
            <label className="inline-label">
              Show category
              <select value={categoryFilter} onChange={(e) => setCategoryFilter(e.target.value)}>
                <option value="">All categories</option>
                {CATEGORIES.map((c) => (
                  <option key={c.value} value={c.value}>{c.label}</option>
                ))}
              </select>
            </label>
          </div>

          <div className="legend">
            {CATEGORIES.map((c) => (
              <span key={c.value}>
                <i style={{ background: c.color }} /> {c.label}
              </span>
            ))}
            <span><i style={{ background: "#7a5af8" }} /> Drawing</span>
          </div>

          <BaseMap>
            <FitToGeofences geofences={geofences} />
            <GeofenceLayer geofences={geofences} interactive={false} />
            <MapClick onClick={addPoint} />
            {points.length >= 3 && <Polygon positions={points} pathOptions={DRAFT_STYLE} />}
            {points.length === 2 && <Polyline positions={points} pathOptions={DRAFT_STYLE} />}
            {points.map((point, index) => (
              <CircleMarker key={index} center={point} radius={5}
                pathOptions={{ color: "#7a5af8", fillOpacity: 1, interactive: false }} />
            ))}
          </BaseMap>
        </section>

        <section className="card">
          <h2>All geofences</h2>
          {loading && <p className="muted">Loading geofences...</p>}
          {loadError && <p className="error">Could not load geofences: {loadError}</p>}
          {!loading && !loadError && geofences.length === 0 && (
            <p className="muted">
              {categoryFilter
                ? "No geofences in this category."
                : "No geofences yet. Draw one on the map to get started."}
            </p>
          )}
          {geofences.length > 0 && (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr><th>Name</th><th>Category</th><th>Corners</th><th>Description</th><th>Created</th></tr>
                </thead>
                <tbody>
                  {geofences.map((g) => (
                    <tr key={g.id}>
                      <td>{g.name}</td>
                      <td>
                        <span className="badge badge-cat" style={{ background: categoryColor(g.category) }}>
                          {categoryLabel(g.category)}
                        </span>
                      </td>
                      <td>{Math.max(g.coordinates.length - 1, 0)}</td>
                      <td>{g.description || "-"}</td>
                      <td>{formatDateTime(g.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
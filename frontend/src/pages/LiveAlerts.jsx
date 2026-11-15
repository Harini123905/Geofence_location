import { useAlerts } from "../alerts-context";

export default function LiveAlerts() {
  const { alerts, status } = useAlerts();

  return (
    <section className="card">
      <h2>Live alerts</h2>
      {alerts.length === 0 ? (
        <p className="muted">
          {status === "connected"
            ? "Waiting for alerts. They appear here as soon as a vehicle enters or exits a geofence."
            : "Connecting to the alert stream..."}
        </p>
      ) : (
        <ul className="feed">
          {alerts.map((alert) => (
            <li
              key={alert.event_id}
              className={alert.geofence.category === "restricted_zone" ? "feed-item danger" : "feed-item"}
            >
              <div>
                <span className={`badge badge-${alert.event_type}`}>{alert.event_type}</span>{" "}
                <strong>{alert.vehicle.vehicle_number}</strong> ({alert.vehicle.driver_name})
              </div>
              <div>
                {alert.geofence.geofence_name}{" "}
                <span className="muted">[{alert.geofence.category}]</span>
              </div>
              <div className="muted">
                {new Date(alert.timestamp).toLocaleString()} · {alert.location.latitude},{" "}
                {alert.location.longitude}
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
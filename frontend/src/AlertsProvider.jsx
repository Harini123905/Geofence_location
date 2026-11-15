import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { WS_URL } from "./api";
import { AlertsContext } from "./alerts-context";

const MAX_ALERTS = 100;

function showToast(alert) {
  const action = alert.event_type === "entry" ? "entered" : "exited";
  const message = `${alert.vehicle.vehicle_number} ${action} ${alert.geofence.geofence_name}`;
  const options = { duration: 6000 };

  if (alert.geofence.category === "restricted_zone") {
    toast.error(`Restricted zone: ${message}`, options);
  } else {
    toast(message, { ...options, icon: alert.event_type === "entry" ? "📍" : "🚪" });
  }
}

// Keeps one WebSocket open for the whole app and shares the alerts it receives.
export default function AlertsProvider({ children }) {
  const [alerts, setAlerts] = useState([]);
  const [status, setStatus] = useState("connecting");

  useEffect(() => {
    let socket;
    let retryTimer;
    let stopped = false;

    function connect() {
      socket = new WebSocket(WS_URL);
      socket.onopen = () => setStatus("connected");
      socket.onmessage = (event) => {
        const alert = JSON.parse(event.data);
        setAlerts((previous) => [alert, ...previous].slice(0, MAX_ALERTS));
        showToast(alert);
      };
      socket.onclose = () => {
        if (stopped) return;
        setStatus("reconnecting");
        retryTimer = setTimeout(connect, 3000);
      };
    }

    connect();
    return () => {
      stopped = true;
      clearTimeout(retryTimer);
      socket?.close();
    };
  }, []);

  return (
    <AlertsContext.Provider value={{ alerts, status }}>
      {children}
    </AlertsContext.Provider>
  );
}
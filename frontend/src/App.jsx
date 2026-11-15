import { NavLink, Navigate, Route, Routes } from "react-router-dom";
import { Toaster } from "react-hot-toast";
import { useAlerts } from "./alerts-context";
import ErrorBoundary from "./components/ErrorBoundary";
import Vehicles from "./pages/Vehicles";
import Geofences from "./pages/Geofences";
import UpdateLocation from "./pages/UpdateLocation";
import AlertRules from "./pages/AlertRules";
import History from "./pages/History";
import LiveAlerts from "./pages/LiveAlerts";

export default function App() {
  const { status } = useAlerts();

  return (
    <>
      <header className="topbar">
        <h1>Geofence Tracker</h1>
        <span className={`status status-${status}`}>
          <span className="dot" /> Live: {status}
        </span>
        <nav>
          <NavLink to="/vehicles">Vehicles</NavLink>
          <NavLink to="/geofences">Geofences</NavLink>
          <NavLink to="/location">Update Location</NavLink>
          <NavLink to="/rules">Alert Rules</NavLink>
          <NavLink to="/history">History</NavLink>
          <NavLink to="/alerts">Live Alerts</NavLink>
        </nav>
      </header>

      <main className="page">
        <ErrorBoundary>
          <Routes>
            <Route path="/" element={<Navigate to="/vehicles" replace />} />
            <Route path="/vehicles" element={<Vehicles />} />
            <Route path="/geofences" element={<Geofences />} />
            <Route path="/location" element={<UpdateLocation />} />
            <Route path="/rules" element={<AlertRules />} />
            <Route path="/history" element={<History />} />
            <Route path="/alerts" element={<LiveAlerts />} />
            <Route
              path="*"
              element={
                <section className="card">
                  <h2>Page not found</h2>
                  <p><NavLink to="/vehicles">Go back to Vehicles</NavLink></p>
                </section>
              }
            />
          </Routes>
        </ErrorBoundary>
      </main>

      <Toaster position="top-right" />
    </>
  );
}
import { createContext, useContext } from "react";

export const AlertsContext = createContext({ alerts: [], status: "connecting" });

export function useAlerts() {
  return useContext(AlertsContext);
}
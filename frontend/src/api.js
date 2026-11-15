export const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";
export const WS_URL = API_URL.replace(/^http/, "ws") + "/ws/alerts";

// Calls the backend and turns error responses into readable errors.
export async function api(path, options = {}) {
  const response = await fetch(`${API_URL}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  let data = null;
  try {
    data = await response.json();
  } catch {
    // The response had no JSON body.
  }

  if (!response.ok) {
    throw new Error(data?.error || `Request failed (${response.status})`);
  }
  return data;
}
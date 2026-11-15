export const CATEGORIES = [
  { value: "delivery_zone", label: "Delivery zone", color: "#2f6fed" },
  { value: "restricted_zone", label: "Restricted zone", color: "#d92d20" },
  { value: "toll_zone", label: "Toll zone", color: "#f79009" },
  { value: "customer_area", label: "Customer area", color: "#12b76a" },
];

export function categoryColor(category) {
  return CATEGORIES.find((c) => c.value === category)?.color || "#667085";
}

export function categoryLabel(category) {
  return CATEGORIES.find((c) => c.value === category)?.label || category;
}

// Map starts here until geofences are loaded.
export const DEFAULT_CENTER = [37.78, -122.41];
export const DEFAULT_ZOOM = 12;
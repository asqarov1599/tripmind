const BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api";

// ─── Core fetcher ────────────────────────────────────────────────────────────
async function request(endpoint, options = {}) {
  const url = `${BASE_URL}${endpoint}`;
  const response = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Request failed: ${response.status}`);
  }

  return response.json();
}

// ─── API Methods ─────────────────────────────────────────────────────────────

/**
 * Search for available flights and hotels
 * @param {Object} payload - Search parameters
 * @param {string} payload.origin - Origin airport code (e.g., "TAS")
 * @param {string} payload.destination - Destination airport code (e.g., "IST")
 * @param {string} payload.departure_date - ISO date string
 * @param {string} payload.return_date - ISO date string
 * @param {number} payload.budget - Total budget in USD
 * @param {number} payload.passengers - Number of passengers
 */
export async function searchFlightsAndHotels(payload) {
  return request("/search", {
    method: "POST",
    body: JSON.stringify({
      ...payload,
      budget: Number(payload.budget),
      passengers: Number(payload.passengers),
    }),
  });
}

/**
 * Generate a PDF itinerary
 * @param {Object} payload
 * @param {string} payload.search_id
 * @param {number} payload.selected_flight_index
 * @param {number} payload.selected_hotel_index
 * @param {string} payload.traveler_name
 */
export async function generateItinerary(payload) {
  return request("/generate", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

/**
 * Download a generated PDF itinerary
 * @param {string} id - Itinerary ID
 */
export function downloadItineraryPDF(id) {
  const a = document.createElement("a");
  a.href = `${BASE_URL}/download/${id}`;
  a.download = "tripmind-itinerary.pdf";
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}

/**
 * Health check endpoint
 */
export async function healthCheck() {
  return request("/health");
}
import { useState } from "react";
import { searchFlightsAndHotels } from "../services/api";
import "./Home.css";

const POPULAR_ROUTES = [
  { origin: "TAS", dest: "IST" },
  { origin: "TAS", dest: "DXB" },
  { origin: "TAS", dest: "CDG" },
  { origin: "TAS", dest: "LHR" },
];

function validate(form) {
  if (!form.origin || form.origin.length < 3)
    return "Enter a valid origin airport code (3 letters).";
  if (!form.destination || form.destination.length < 3)
    return "Enter a valid destination airport code (3 letters).";
  if (!form.departure_date)
    return "Please select a departure date.";
  if (!form.return_date)
    return "Please select a return date.";
  if (form.return_date <= form.departure_date)
    return "Return date must be after departure date.";
  if (!form.budget || Number(form.budget) <= 0)
    return "Enter a valid budget amount (USD).";
  return null;
}

export default function Home({ onResults }) {
  const [form, setForm] = useState({
    origin: "",
    destination: "",
    departure_date: "",
    return_date: "",
    budget: "",
    passengers: "1",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const set = (key) => (e) =>
    setForm((prev) => ({ ...prev, [key]: e.target.value }));

  const fillRoute = (origin, dest) =>
    setForm((prev) => ({ ...prev, origin, destination: dest }));

  const handleSearch = async () => {
    const err = validate(form);
    if (err) {
      setError(err);
      return;
    }
    setError(null);
    setLoading(true);
    try {
      const data = await searchFlightsAndHotels(form);
      onResults(data, form);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="home page-enter">
      {/* ── Hero ────────────────────────────────────────────── */}
      <section className="home__hero">
        <div className="home__hero-bg" aria-hidden="true">
          <div className="home__hero-orb home__hero-orb--1" />
          <div className="home__hero-orb home__hero-orb--2" />
          <div className="home__hero-grid" />
        </div>

        <div className="home__hero-content">
          <div className="home__hero-eyebrow animate-fadeUp">
            <span className="home__hero-badge">✦ AI-Powered</span>
          </div>
          <h1 className="home__hero-heading heading-display animate-fadeUp delay-1">
            Plan smarter.
            <br />
            <em className="home__hero-accent">Travel better.</em>
          </h1>
          <p className="home__hero-sub animate-fadeUp delay-2">
            Flight & hotel search with personalised AI recommendations
            <br />and one-click itinerary PDFs.
          </p>
        </div>
      </section>

      {/* ── Search Card ─────────────────────────────────────── */}
      <div className="home__search-wrap">
        <div className="home__search-card animate-scaleIn delay-3">
          <div className="home__search-header">
            <span className="home__search-icon">✈</span>
            <h2>Where would you like to go?</h2>
          </div>

          {/* Row 1: Route + Passengers */}
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">From</label>
              <input
                className="form-input"
                placeholder="e.g. TAS"
                value={form.origin}
                onChange={set("origin")}
                maxLength={3}
                style={{ textTransform: "uppercase" }}
              />
            </div>

            <div className="home__swap-col">
              <div className="home__swap-line" />
              <button
                className="home__swap-btn"
                onClick={() =>
                  setForm((prev) => ({
                    ...prev,
                    origin: prev.destination,
                    destination: prev.origin,
                  }))
                }
                title="Swap airports"
              >
                ⇄
              </button>
              <div className="home__swap-line" />
            </div>

            <div className="form-group">
              <label className="form-label">To</label>
              <input
                className="form-input"
                placeholder="e.g. IST"
                value={form.destination}
                onChange={set("destination")}
                maxLength={3}
                style={{ textTransform: "uppercase" }}
              />
            </div>

            <div className="form-group home__pax-group">
              <label className="form-label">Passengers</label>
              <select className="form-select" value={form.passengers} onChange={set("passengers")}>
                {[1, 2, 3, 4, 5, 6].map((n) => (
                  <option key={n} value={n}>{n} {n === 1 ? "Traveler" : "Travelers"}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Row 2: Dates + Budget */}
          <div className="form-row" style={{ marginTop: 16 }}>
            <div className="form-group">
              <label className="form-label">Departure</label>
              <input
                className="form-input"
                type="date"
                value={form.departure_date}
                onChange={set("departure_date")}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Return</label>
              <input
                className="form-input"
                type="date"
                value={form.return_date}
                onChange={set("return_date")}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Budget (USD)</label>
              <input
                className="form-input"
                type="number"
                placeholder="e.g. 1500"
                value={form.budget}
                onChange={set("budget")}
                min={1}
              />
            </div>
          </div>

          {error && (
            <div className="error-box" style={{ marginTop: 16 }}>
              ⚠ {error}
            </div>
          )}

          <button
            className="btn btn--gold btn--full"
            style={{ marginTop: 24 }}
            onClick={handleSearch}
            disabled={loading}
          >
            {loading ? (
              <>
                <span className="spinner spinner--sm" />
                Searching…
              </>
            ) : (
              "Search Flights & Hotels →"
            )}
          </button>

          {/* Popular routes */}
          <div className="home__popular">
            <span className="home__popular-label">Popular:</span>
            {POPULAR_ROUTES.map(({ origin, dest }) => (
              <button
                key={`${origin}-${dest}`}
                className="home__popular-btn"
                onClick={() => fillRoute(origin, dest)}
              >
                {origin} → {dest}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* ── Stats strip ─────────────────────────────────────── */}
      <div className="home__stats animate-fadeUp delay-5">
        {[
          { value: "50+", label: "Routes" },
          { value: "100+", label: "Hotels" },
          { value: "< 2s", label: "Search time" },
          { value: "AI", label: "Recommendations" },
        ].map(({ value, label }) => (
          <div key={label} className="home__stat">
            <span className="home__stat-value">{value}</span>
            <span className="home__stat-label">{label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
import { useState } from "react";
import { searchFlightsAndHotels } from "../services/api";
import "./Home.css";

const POPULAR_ROUTES = [
  { origin: "TAS", dest: "IST", label: "Istanbul" },
  { origin: "TAS", dest: "DXB", label: "Dubai" },
  { origin: "TAS", dest: "CDG", label: "Paris" },
  { origin: "TAS", dest: "LHR", label: "London" },
];

const PlaneIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
    <path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"/>
  </svg>
);

const SwapIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M7 16V4m0 0L3 8m4-4 4 4M17 8v12m0 0 4-4m-4 4-4-4"/>
  </svg>
);

const AlertIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
  </svg>
);

const SearchIcon = () => (
  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
  </svg>
);

function validate(form) {
  if (!form.origin || form.origin.length < 3) return "Enter a valid origin airport code (3 letters).";
  if (!form.destination || form.destination.length < 3) return "Enter a valid destination airport code (3 letters).";
  if (!form.departure_date) return "Please select a departure date.";
  if (!form.return_date) return "Please select a return date.";
  if (form.return_date <= form.departure_date) return "Return date must be after departure date.";
  if (!form.budget || Number(form.budget) <= 0) return "Enter a valid budget amount (USD).";
  return null;
}

export default function Home({ onResults }) {
  const [form, setForm] = useState({ origin: "", destination: "", departure_date: "", return_date: "", budget: "", passengers: "1" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const set = (key) => (e) => setForm((prev) => ({ ...prev, [key]: e.target.value }));
  const fillRoute = (origin, dest) => setForm((prev) => ({ ...prev, origin, destination: dest }));

  const handleSearch = async () => {
    const err = validate(form);
    if (err) { setError(err); return; }
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
      <section className="home__hero">
        <div className="home__hero-bg" aria-hidden="true">
          <div className="home__hero-glow" />
          <div className="home__hero-texture" />
          <div className="home__hero-dot home__hero-dot--1" />
          <div className="home__hero-dot home__hero-dot--2" />
          <div className="home__hero-dot home__hero-dot--3" />
        </div>
        <div className="home__hero-content">
          <div className="home__hero-eyebrow animate-fadeUp">
            <span className="home__hero-badge">
              <PlaneIcon />
              Smart travel planning
            </span>
          </div>
          <h1 className="home__hero-heading heading-display animate-fadeUp delay-1">
            Your next trip,<br />
            <em className="home__hero-accent">thoughtfully planned.</em>
          </h1>
          <p className="home__hero-sub animate-fadeUp delay-2">
            Search real flights and hotels, get personalised recommendations,<br />
            and download your itinerary — all in one place.
          </p>
        </div>
      </section>

      <div className="home__search-wrap">
        <div className="home__search-card animate-scaleIn delay-3">
          <div className="home__search-header">
            <div className="home__search-icon-wrap"><PlaneIcon /></div>
            <h2>Where would you like to go?</h2>
          </div>

          <div className="form-row">
            <div className="form-group">
              <label className="form-label">From</label>
              <input className="form-input" placeholder="e.g. TAS" value={form.origin} onChange={set("origin")} maxLength={3} style={{ textTransform: "uppercase" }} />
            </div>
            <div className="home__swap-col">
              <div className="home__swap-line" />
              <button className="home__swap-btn" onClick={() => setForm((prev) => ({ ...prev, origin: prev.destination, destination: prev.origin }))} title="Swap airports">
                <SwapIcon />
              </button>
              <div className="home__swap-line" />
            </div>
            <div className="form-group">
              <label className="form-label">To</label>
              <input className="form-input" placeholder="e.g. IST" value={form.destination} onChange={set("destination")} maxLength={3} style={{ textTransform: "uppercase" }} />
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

          <div className="form-row" style={{ marginTop: 16 }}>
            <div className="form-group">
              <label className="form-label">Departure</label>
              <input className="form-input" type="date" value={form.departure_date} onChange={set("departure_date")} />
            </div>
            <div className="form-group">
              <label className="form-label">Return</label>
              <input className="form-input" type="date" value={form.return_date} onChange={set("return_date")} />
            </div>
            <div className="form-group">
              <label className="form-label">Budget (USD)</label>
              <input className="form-input" type="number" placeholder="e.g. 1500" value={form.budget} onChange={set("budget")} min={1} />
            </div>
          </div>

          {error && (
            <div className="error-box" style={{ marginTop: 16 }}>
              <AlertIcon /> {error}
            </div>
          )}

          <button className="btn btn--gold btn--full" style={{ marginTop: 24 }} onClick={handleSearch} disabled={loading}>
            {loading ? (
              <><span className="spinner spinner--sm" /> Searching…</>
            ) : (
              <><SearchIcon /> Search Flights &amp; Hotels</>
            )}
          </button>

          <div className="home__popular">
            <span className="home__popular-label">Popular from Tashkent:</span>
            {POPULAR_ROUTES.map(({ origin, dest, label }) => (
              <button key={`${origin}-${dest}`} className="home__popular-btn" onClick={() => fillRoute(origin, dest)}>
                {label}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="home__stats animate-fadeUp delay-5">
        {[
          { icon: "✈", value: "Real flights", label: "Live airline data" },
          { icon: "🏨", value: "Top hotels", label: "Curated picks" },
          { icon: "💡", value: "AI advice", label: "Budget-aware tips" },
          { icon: "📋", value: "PDF ready", label: "One-click itinerary" },
        ].map(({ icon, value, label }) => (
          <div key={label} className="home__stat">
            <span className="home__stat-icon">{icon}</span>
            <span className="home__stat-value">{value}</span>
            <span className="home__stat-label">{label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

import { useState } from "react";
import { generateItinerary, downloadItineraryPDF } from "../services/api";
import Stars from "../components/Stars";
import "./Results.css";

function fmtDate(iso) {
  if (!iso) return "—";
  return new Date(iso + "T00:00:00").toLocaleDateString("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function fmtTime(iso) {
  if (!iso) return "—";
  return new Date(iso).toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: true,
  });
}

function FlightCard({ flight, index, selected, onSelect }) {
  return (
    <article
      className={`result-card result-card--flight ${selected ? "card--selected" : ""} card--interactive`}
      onClick={() => onSelect(index)}
    >
      <div className="flight-card__check">{selected && <span>✓</span>}</div>

      <div className="flight-card__airline">
        <div className="flight-card__airline-name">{flight.airline}</div>
        <span className={`badge ${flight.stops === 0 ? "badge--direct" : "badge--stop"}`}>
          {flight.stops === 0 ? "Direct" : `${flight.stops} stop${flight.stops > 1 ? "s" : ""}`}
        </span>
      </div>

      <div className="flight-card__route">
        <div className="flight-card__leg">
          <span className="flight-card__leg-label">Outbound</span>
          <span className="flight-card__leg-time">{fmtTime(flight.departure_time)}</span>
          <span className="flight-card__leg-arrow">→</span>
          <span className="flight-card__leg-time">{fmtTime(flight.arrival_time)}</span>
          <span className="flight-card__leg-dur">{flight.duration}</span>
        </div>
        <div className="flight-card__leg">
          <span className="flight-card__leg-label">Return</span>
          <span className="flight-card__leg-time">{fmtTime(flight.return_departure_time)}</span>
          <span className="flight-card__leg-arrow">→</span>
          <span className="flight-card__leg-time">{fmtTime(flight.return_arrival_time)}</span>
          <span className="flight-card__leg-dur">{flight.return_duration}</span>
        </div>
      </div>

      <div className="flight-card__price">
        <div className="result-card__price-amount">${flight.price.toLocaleString()}</div>
        <div className="result-card__price-label">per person</div>
      </div>
    </article>
  );
}

function HotelCard({ hotel, index, selected, onSelect }) {
  return (
    <article
      className={`result-card result-card--hotel ${selected ? "card--selected" : ""} card--interactive`}
      onClick={() => onSelect(index)}
    >
      <div className="result-card__check">{selected && <span>✓</span>}</div>

      <div className="hotel-card__info">
        <div className="hotel-card__name">{hotel.name}</div>
        <div className="hotel-card__location">
          <span className="hotel-card__location-icon">📍</span>
          {hotel.location}
        </div>
        <div className="hotel-card__rating">
          <Stars rating={hotel.rating} />
        </div>
      </div>

      <div className="result-card__price-block">
        <div className="result-card__price-amount">${hotel.price.toLocaleString()}</div>
        <div className="result-card__price-label">/ night</div>
      </div>
    </article>
  );
}

export default function Results({ data, searchForm, onBack }) {
  const [selFlight, setSelFlight] = useState(0);
  const [selHotel, setSelHotel] = useState(0);
  const [travelerName, setTravelerName] = useState("");
  const [generating, setGenerating] = useState(false);
  const [genError, setGenError] = useState(null);
  const [pdfId, setPdfId] = useState(null);

  const depD = new Date(searchForm.departure_date + "T00:00:00");
  const retD = new Date(searchForm.return_date + "T00:00:00");
  const nights = Math.round((retD - depD) / 86400000);

  const flight = data.flights[selFlight];
  const hotel = data.hotels[selHotel];
  const totalCost = flight.price + hotel.price * nights;

  const handleGenerate = async () => {
    setGenError(null);
    setGenerating(true);
    setPdfId(null);
    try {
      const res = await generateItinerary({
        search_id: data.search_id,
        selected_flight_index: selFlight,
        selected_hotel_index: selHotel,
        traveler_name: travelerName || "Guest Traveler",
      });
      setPdfId(res.itinerary_id);
    } catch (e) {
      setGenError(e.message);
    } finally {
      setGenerating(false);
    }
  };

  return (
    <div className="results section page-enter">
      {/* ── Breadcrumb ───────────────────────────────────────── */}
      <div className="results__header">
        <button className="btn btn--ghost btn--sm" onClick={onBack}>
          ← Back to Search
        </button>

        <div className="results__route">
          <span className="tag">{searchForm.origin.toUpperCase()}</span>
          <span className="results__route-arrow">→</span>
          <span className="tag">{searchForm.destination.toUpperCase()}</span>
          <span className="results__route-meta">
            {fmtDate(searchForm.departure_date)} – {fmtDate(searchForm.return_date)}
            &nbsp;·&nbsp;
            {searchForm.passengers} pax
          </span>
        </div>
      </div>

      {/* ── AI Summary ───────────────────────────────────────── */}
      {data.ai_summary && (
        <div className="ai-box animate-scaleIn">
          <div className="ai-box__header">
            <div className="ai-box__icon">✦</div>
            <span className="ai-box__title">AI Recommendations</span>
          </div>
          <p className="ai-box__body">{data.ai_summary}</p>
        </div>
      )}

      {/* ── Flights ──────────────────────────────────────────── */}
      <div className="results__section animate-fadeUp delay-1">
        <div className="results__section-head">
          <h2 className="heading-section">Flights</h2>
          <span className="text-label">Round-trip · per person · select one</span>
        </div>
        <div className="results__list">
          {data.flights.map((f, i) => (
            <FlightCard
              key={i}
              flight={f}
              index={i}
              selected={selFlight === i}
              onSelect={setSelFlight}
            />
          ))}
        </div>
      </div>

      {/* ── Hotels ───────────────────────────────────────────── */}
      <div className="results__section animate-fadeUp delay-2">
        <div className="results__section-head">
          <h2 className="heading-section">Hotels</h2>
          <span className="text-label">Price per night · select one</span>
        </div>
        <div className="results__list">
          {data.hotels.map((h, i) => (
            <HotelCard
              key={i}
              hotel={h}
              index={i}
              selected={selHotel === i}
              onSelect={setSelHotel}
            />
          ))}
        </div>
      </div>

      {/* ── Confirm Panel ────────────────────────────────────── */}
      <div className="confirm-panel animate-fadeUp delay-3">
        <h3 className="confirm-panel__title">Your Itinerary</h3>

        <div className="confirm-panel__rows">
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">✈ Flight</span>
            <span className="confirm-panel__row-value">
              {flight.airline} &mdash; ${flight.price.toLocaleString()}
            </span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">🏨 Hotel</span>
            <span className="confirm-panel__row-value">
              {hotel.name} &mdash; ${hotel.price.toLocaleString()}/night
            </span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">🌙 Nights</span>
            <span className="confirm-panel__row-value">{nights}</span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">🏨 Hotel total</span>
            <span className="confirm-panel__row-value">
              ${(hotel.price * nights).toLocaleString()}
            </span>
          </div>
        </div>

        <div className="confirm-panel__total">
          <span className="confirm-panel__total-label">Estimated Total</span>
          <span className="confirm-panel__total-value">${totalCost.toLocaleString()}</span>
        </div>

        {/* Traveler Name */}
        <div className="form-group" style={{ marginTop: 24 }}>
          <label className="form-label">Traveler Name (optional)</label>
          <input
            className="form-input"
            placeholder="e.g. Jane Doe"
            value={travelerName}
            onChange={(e) => setTravelerName(e.target.value)}
          />
        </div>

        {genError && (
          <div className="error-box" style={{ marginTop: 16 }}>
            ⚠ {genError}
          </div>
        )}
        {pdfId && (
          <div className="success-box" style={{ marginTop: 16 }}>
            ✅ PDF generated successfully!
          </div>
        )}

        <div className="confirm-panel__actions">
          <button
            className="btn btn--gold"
            onClick={handleGenerate}
            disabled={generating}
          >
            {generating ? (
              <>
                <span className="spinner spinner--sm" />
                Generating…
              </>
            ) : (
              "📄 Generate PDF"
            )}
          </button>
          {pdfId && (
            <button
              className="btn btn--navy"
              onClick={() => downloadItineraryPDF(pdfId)}
            >
              ⬇ Download PDF
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
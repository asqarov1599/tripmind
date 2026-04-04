import { useState } from "react";
import { generateItinerary, downloadItineraryPDF } from "../services/api";
import Stars from "../components/Stars";
import "./Results.css";

/* ─── Helpers ─────────────────────────────────────────────────────────────── */

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

function fmtPrice(n) {
  return Number(n).toLocaleString("en-US", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  });
}

/**
 * Render **bold** markdown as <strong> and \n\n as paragraphs.
 * Strips any remaining double-asterisks that aren't paired.
 */
function renderMarkdown(text) {
  if (!text) return null;

  // Split on double newlines into paragraphs
  const paragraphs = text.split(/\n\n+/);

  return paragraphs.map((para, pi) => {
    // Split on **…** bold markers
    const parts = para.split(/\*\*(.+?)\*\*/g);
    const nodes = parts.map((part, i) =>
      i % 2 === 1 ? <strong key={i}>{part}</strong> : part
    );
    return (
      <p key={pi} className="ai-para">
        {nodes}
      </p>
    );
  });
}

/* ─── Flight Card ─────────────────────────────────────────────────────────── */

function FlightCard({ flight, index, selected, onSelect }) {
  const isNaN_price = isNaN(flight.price) || flight.price <= 0;

  return (
    <article
      className={`result-card result-card--flight ${selected ? "card--selected" : ""}`}
      onClick={() => onSelect(index)}
      role="button"
      aria-pressed={selected}
    >
      {selected && <div className="card__check-badge">✓</div>}

      {/* Airline + stop badge */}
      <div className="fc__airline-col">
        <div className="fc__airline-name">{flight.airline}</div>
        {flight.flight_number && (
          <div className="fc__flight-num">{flight.flight_number}</div>
        )}
        <span className={`badge ${flight.stops === 0 ? "badge--direct" : "badge--stop"}`}>
          {flight.stops === 0 ? "Direct" : `${flight.stops} stop${flight.stops > 1 ? "s" : ""}`}
        </span>
      </div>

      {/* Outbound + return legs */}
      <div className="fc__legs">
        <div className="fc__leg">
          <span className="fc__leg-tag">Out</span>
          <span className="fc__time">{fmtTime(flight.departure_time)}</span>
          <span className="fc__arrow">→</span>
          <span className="fc__time">{fmtTime(flight.arrival_time)}</span>
          <span className="fc__dur">{flight.duration}</span>
        </div>
        <div className="fc__leg">
          <span className="fc__leg-tag">Ret</span>
          <span className="fc__time">{fmtTime(flight.return_departure_time)}</span>
          <span className="fc__arrow">→</span>
          <span className="fc__time">{fmtTime(flight.return_arrival_time)}</span>
          <span className="fc__dur">{flight.return_duration}</span>
        </div>
      </div>

      {/* Price */}
      <div className="fc__price-col">
        {isNaN_price ? (
          <div className="price-na">N/A</div>
        ) : (
          <>
            <div className="price-amount">${fmtPrice(flight.price)}</div>
            <div className="price-label">per person</div>
          </>
        )}
      </div>
    </article>
  );
}

/* ─── Hotel Card ──────────────────────────────────────────────────────────── */

/**
 * Amadeus sometimes returns total-stay price instead of per-night.
 * If the price looks unreasonably high (> $2000/night) we show a warning
 * and skip it from the summary calculation — but we still display it.
 */
function sanitizeHotelPrice(price) {
  const p = Number(price);
  if (isNaN(p) || p <= 0) return null;
  return p;
}

function HotelCard({ hotel, index, selected, onSelect, nights }) {
  const price = sanitizeHotelPrice(hotel.price);
  const suspicious = price && price > 1500;

  return (
    <article
      className={`result-card result-card--hotel ${selected ? "card--selected" : ""}`}
      onClick={() => onSelect(index)}
      role="button"
      aria-pressed={selected}
    >
      {selected && <div className="card__check-badge">✓</div>}

      <div className="hc__info">
        <div className="hc__name">{hotel.name}</div>
        <div className="hc__location">
          <span className="hc__pin">📍</span>
          {hotel.location}
        </div>
        <div className="hc__stars">
          <Stars rating={hotel.rating} />
          <span className="hc__rating-num">{hotel.rating?.toFixed(1)}</span>
        </div>
        {suspicious && (
          <div className="hc__price-warn">
            ⚠ Price may reflect total stay, not per night
          </div>
        )}
      </div>

      <div className="hc__price-col">
        {price ? (
          <>
            <div className="price-amount">${fmtPrice(price)}</div>
            <div className="price-label">/ night</div>
            {nights && (
              <div className="price-total-hint">
                ${fmtPrice(price * nights)} total
              </div>
            )}
          </>
        ) : (
          <div className="price-na">N/A</div>
        )}
      </div>
    </article>
  );
}

/* ─── Main Results Page ───────────────────────────────────────────────────── */

export default function Results({ data, searchForm, onBack }) {
  const [selFlight, setSelFlight] = useState(0);
  const [selHotel, setSelHotel]   = useState(0);
  const [travelerName, setTravelerName] = useState("");
  const [generating, setGenerating]     = useState(false);
  const [genError, setGenError]         = useState(null);
  const [pdfId, setPdfId]               = useState(null);

  const depD   = new Date(searchForm.departure_date + "T00:00:00");
  const retD   = new Date(searchForm.return_date    + "T00:00:00");
  const nights = Math.round((retD - depD) / 86400000);

  const flight = data.flights?.[selFlight];
  const hotel  = data.hotels?.[selHotel];

  const flightPrice = flight ? Number(flight.price) || 0 : 0;
  const hotelPrice  = hotel  ? sanitizeHotelPrice(hotel.price) || 0 : 0;
  const totalCost   = flightPrice + hotelPrice * nights;

  const handleGenerate = async () => {
    setGenError(null);
    setGenerating(true);
    setPdfId(null);
    try {
      const res = await generateItinerary({
        search_id:             data.search_id,
        selected_flight_index: selFlight,
        selected_hotel_index:  selHotel,
        traveler_name:         travelerName || "Guest Traveler",
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

      {/* ── Breadcrumb ─────────────────────────────────────── */}
      <div className="results__header">
        <button className="btn btn--ghost btn--sm" onClick={onBack}>
          ← Back
        </button>
        <div className="results__route">
          <span className="route-chip">{searchForm.origin?.toUpperCase()}</span>
          <span className="route-arrow">→</span>
          <span className="route-chip">{searchForm.destination?.toUpperCase()}</span>
          <span className="route-meta">
            {fmtDate(searchForm.departure_date)} – {fmtDate(searchForm.return_date)}
            &nbsp;·&nbsp;{searchForm.passengers} pax
            &nbsp;·&nbsp;{nights} nights
          </span>
        </div>
      </div>

      {/* ── AI Summary ─────────────────────────────────────── */}
      {data.ai_summary && (
        <div className="ai-box">
          <div className="ai-box__header">
            <div className="ai-box__icon">✦</div>
            <span className="ai-box__title">AI Recommendations</span>
            {data.source === "estimated" && (
              <span className="ai-box__badge">Estimated data</span>
            )}
          </div>
          <div className="ai-box__body">
            {renderMarkdown(data.ai_summary)}
          </div>
        </div>
      )}

      {/* ── Flights ────────────────────────────────────────── */}
      <div className="results__section">
        <div className="results__section-head">
          <h2 className="heading-section">Flights</h2>
          <span className="text-label">Round-trip · per person · select one</span>
        </div>
        <div className="results__list">
          {data.flights?.map((f, i) => (
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

      {/* ── Hotels ─────────────────────────────────────────── */}
      <div className="results__section">
        <div className="results__section-head">
          <h2 className="heading-section">Hotels</h2>
          <span className="text-label">Price per night · select one</span>
        </div>
        <div className="results__list">
          {data.hotels?.map((h, i) => (
            <HotelCard
              key={i}
              hotel={h}
              index={i}
              selected={selHotel === i}
              onSelect={setSelHotel}
              nights={nights}
            />
          ))}
        </div>
      </div>

      {/* ── Confirm Panel ──────────────────────────────────── */}
      <div className="confirm-panel">
        <h3 className="confirm-panel__title">Your Selection</h3>

        <div className="confirm-panel__rows">
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">✈ Flight</span>
            <span className="confirm-panel__row-value">
              {flight?.airline} — ${fmtPrice(flightPrice)}/person
            </span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">🏨 Hotel</span>
            <span className="confirm-panel__row-value">
              {hotel?.name}
            </span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">💰 Hotel rate</span>
            <span className="confirm-panel__row-value">
              ${fmtPrice(hotelPrice)} / night
            </span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">🌙 Nights</span>
            <span className="confirm-panel__row-value">{nights}</span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">🏨 Hotel total</span>
            <span className="confirm-panel__row-value">
              ${fmtPrice(hotelPrice * nights)}
            </span>
          </div>
        </div>

        <div className="confirm-panel__total">
          <span className="confirm-panel__total-label">Estimated Total</span>
          <span className="confirm-panel__total-value">${fmtPrice(totalCost)}</span>
        </div>

        <div className="form-group" style={{ marginTop: 24 }}>
          <label className="form-label">Traveler Name <span className="form-label--opt">(optional)</span></label>
          <input
            className="form-input"
            placeholder="e.g. Jane Doe"
            value={travelerName}
            onChange={(e) => setTravelerName(e.target.value)}
          />
        </div>

        {genError && (
          <div className="error-box" style={{ marginTop: 16 }}>⚠ {genError}</div>
        )}
        {pdfId && (
          <div className="success-box" style={{ marginTop: 16 }}>✅ PDF generated successfully!</div>
        )}

        <div className="confirm-panel__actions">
          <button className="btn btn--gold" onClick={handleGenerate} disabled={generating}>
            {generating ? (
              <><span className="spinner spinner--sm" /> Generating…</>
            ) : (
              "📄 Generate PDF Itinerary"
            )}
          </button>
          {pdfId && (
            <button className="btn btn--navy" onClick={() => downloadItineraryPDF(pdfId)}>
              ⬇ Download PDF
            </button>
          )}
        </div>
      </div>

    </div>
  );
}

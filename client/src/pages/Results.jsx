import { useState } from "react";
import { generateItinerary, downloadItineraryPDF } from "../services/api";
import Stars from "../components/Stars";
import {
  Check,
  ArrowRight,
  ArrowLeft,
  MapPin,
  AlertTriangle,
  Plane,
  Hotel,
  DollarSign,
  Moon,
  FileText,
  Download,
  CheckCircle,
  Sparkles,
} from "lucide-react";
import "./Results.css";

function fmtDate(iso) {
  if (!iso) return "—";
  return new Date(iso + "T00:00:00").toLocaleDateString("en-US", {
    weekday: "short", month: "short", day: "numeric", year: "numeric",
  });
}

function fmtTime(iso) {
  if (!iso) return "—";
  return new Date(iso).toLocaleTimeString("en-US", {
    hour: "2-digit", minute: "2-digit", hour12: true,
  });
}

function fmtPrice(n) {
  return Number(n).toLocaleString("en-US", {
    minimumFractionDigits: 0, maximumFractionDigits: 0,
  });
}

function renderMarkdown(text) {
  if (!text) return null;
  const paragraphs = text.split(/\n\n+/);
  return paragraphs.map((para, pi) => {
    const parts = para.split(/\*\*(.+?)\*\*/g);
    const nodes = parts.map((part, i) =>
      i % 2 === 1 ? <strong key={i}>{part}</strong> : part
    );
    return <p key={pi} className="ai-para">{nodes}</p>;
  });
}

function FlightCard({ flight, index, selected, onSelect }) {
  const isNaN_price = isNaN(flight.price) || flight.price <= 0;
  return (
    <article
      className={`result-card result-card--flight ${selected ? "card--selected" : ""}`}
      onClick={() => onSelect(index)}
      role="button"
      aria-pressed={selected}
    >
      {selected && (
        <div className="card__check-badge">
          <Check size={11} strokeWidth={3} />
        </div>
      )}
      <div className="fc__airline-col">
        <div className="fc__airline-name">{flight.airline}</div>
        {flight.flight_number && <div className="fc__flight-num">{flight.flight_number}</div>}
        <span className={`badge ${flight.stops === 0 ? "badge--direct" : "badge--stop"}`}>
          {flight.stops === 0 ? "Direct" : `${flight.stops} stop${flight.stops > 1 ? "s" : ""}`}
        </span>
      </div>
      <div className="fc__legs">
        <div className="fc__leg">
          <span className="fc__leg-tag">Out</span>
          <span className="fc__time">{fmtTime(flight.departure_time)}</span>
          <span className="fc__arrow"><ArrowRight size={13} /></span>
          <span className="fc__time">{fmtTime(flight.arrival_time)}</span>
          <span className="fc__dur">{flight.duration}</span>
        </div>
        <div className="fc__leg">
          <span className="fc__leg-tag">Ret</span>
          <span className="fc__time">{fmtTime(flight.return_departure_time)}</span>
          <span className="fc__arrow"><ArrowRight size={13} /></span>
          <span className="fc__time">{fmtTime(flight.return_arrival_time)}</span>
          <span className="fc__dur">{flight.return_duration}</span>
        </div>
      </div>
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
      {selected && (
        <div className="card__check-badge">
          <Check size={11} strokeWidth={3} />
        </div>
      )}
      <div className="hc__info">
        <div className="hc__name">{hotel.name}</div>
        <div className="hc__location">
          <span className="hc__pin"><MapPin size={12} /></span>
          {hotel.location}
        </div>
        <div className="hc__stars">
          <Stars rating={hotel.rating} />
          <span className="hc__rating-num">{hotel.rating?.toFixed(1)}</span>
        </div>
        {suspicious && (
          <div className="hc__price-warn">
            <AlertTriangle size={12} style={{ display: "inline", marginRight: 4 }} />
            Price may reflect total stay, not per night
          </div>
        )}
      </div>
      <div className="hc__price-col">
        {price ? (
          <>
            <div className="price-amount">${fmtPrice(price)}</div>
            <div className="price-label">/ night</div>
            {nights && (
              <div className="price-total-hint">${fmtPrice(price * nights)} total</div>
            )}
          </>
        ) : (
          <div className="price-na">N/A</div>
        )}
      </div>
    </article>
  );
}

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
          <ArrowLeft size={14} /> Back
        </button>
        <div className="results__route">
          <span className="route-chip">{searchForm.origin?.toUpperCase()}</span>
          <span className="route-arrow"><ArrowRight size={16} /></span>
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
            <div className="ai-box__icon">
              <Sparkles size={15} />
            </div>
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
            <FlightCard key={i} flight={f} index={i} selected={selFlight === i} onSelect={setSelFlight} />
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
            <HotelCard key={i} hotel={h} index={i} selected={selHotel === i} onSelect={setSelHotel} nights={nights} />
          ))}
        </div>
      </div>

      {/* ── Confirm Panel ──────────────────────────────────── */}
      <div className="confirm-panel">
        <h3 className="confirm-panel__title">Your Selection</h3>

        <div className="confirm-panel__rows">
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">
              <Plane size={13} style={{ display: "inline", marginRight: 6, verticalAlign: "middle" }} />
              Flight
            </span>
            <span className="confirm-panel__row-value">
              {flight?.airline} — ${fmtPrice(flightPrice)}/person
            </span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">
              <Hotel size={13} style={{ display: "inline", marginRight: 6, verticalAlign: "middle" }} />
              Hotel
            </span>
            <span className="confirm-panel__row-value">{hotel?.name}</span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">
              <DollarSign size={13} style={{ display: "inline", marginRight: 6, verticalAlign: "middle" }} />
              Hotel rate
            </span>
            <span className="confirm-panel__row-value">${fmtPrice(hotelPrice)} / night</span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">
              <Moon size={13} style={{ display: "inline", marginRight: 6, verticalAlign: "middle" }} />
              Nights
            </span>
            <span className="confirm-panel__row-value">{nights}</span>
          </div>
          <div className="confirm-panel__row">
            <span className="confirm-panel__row-label">
              <Hotel size={13} style={{ display: "inline", marginRight: 6, verticalAlign: "middle" }} />
              Hotel total
            </span>
            <span className="confirm-panel__row-value">${fmtPrice(hotelPrice * nights)}</span>
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
          <div className="error-box" style={{ marginTop: 16 }}>
            <AlertTriangle size={15} /> {genError}
          </div>
        )}
        {pdfId && (
          <div className="success-box" style={{ marginTop: 16 }}>
            <CheckCircle size={15} /> PDF generated successfully!
          </div>
        )}

        <div className="confirm-panel__actions">
          <button className="btn btn--gold" onClick={handleGenerate} disabled={generating}>
            {generating ? (
              <><span className="spinner spinner--sm" /> Generating…</>
            ) : (
              <><FileText size={15} /> Generate PDF Itinerary</>
            )}
          </button>
          {pdfId && (
            <button className="btn btn--navy" onClick={() => downloadItineraryPDF(pdfId)}>
              <Download size={15} /> Download PDF
            </button>
          )}
        </div>
      </div>

    </div>
  );
}

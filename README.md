# TripMind

A travel planning app that pulls real flight and hotel data, runs it through an AI for recommendations, and spits out a clean PDF itinerary you can actually use — for visa applications, trip planning, or just keeping yourself organised.

Built with Go on the backend and React on the frontend. Uses the Amadeus API for live flight/hotel data and HuggingFace for the AI summaries. Falls back gracefully to realistic estimated data when neither is configured, so it works out of the box.

---

## What it does

- **Search flights and hotels** — enter your route, dates, budget, and number of travellers. If Amadeus keys are configured you get live data; otherwise the fallback mode generates convincingly realistic estimates based on known routes.
- **Multi-city routing** — heading to Frankfurt but flying home from Paris? Toggle "Multi-city return" and enter a different departure airport for your return leg. Same workflow, no extra complexity.
- **AI recommendations** — a short summary picks the best flight and hotel for your budget, explains why, and includes destination highlights (things to do, places to see) for popular cities.
- **PDF itinerary** — one click generates a formatted PDF with your traveller name, selected flight, hotel, cost breakdown, AI notes, and destination highlights. Useful for Schengen visa applications or just having something to hand your family.

---

## Getting started

### Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL (or you can swap the driver — the schema is simple)

### Backend

```bash
cd backend
cp .env.example .env   # fill in your keys — see below
go mod download
go run main.go
```

The server starts on port `8080`. No Amadeus or HuggingFace keys? It still runs — you just get estimated prices and a built-in recommendation summary instead of live data.

### Frontend

```bash
cd client
npm install
npm run dev
```

Opens on `http://localhost:5173` by default.

---

## Environment variables

```env
# PostgreSQL
DATABASE_URL=postgres://user:pass@localhost:5432/tripmind

# Amadeus (optional — app works without these, uses estimated data)
AMADEUS_CLIENT_ID=your_client_id
AMADEUS_CLIENT_SECRET=your_client_secret
AMADEUS_ENV=test          # "test" for sandbox, "production" for live

# HuggingFace (optional — app uses built-in summary without this)
HUGGINGFACE_API_KEY=your_key
HF_MODEL=mistralai/Mistral-7B-Instruct-v0.3   # default if not set
```

---

## Project layout

```
tripmind/
├── backend/
│   ├── handlers/
│   │   ├── search.go       # POST /api/search — flights + hotels + AI summary
│   │   ├── itinerary.go    # POST /api/generate — creates PDF
│   │   └── download.go     # GET /api/download/:id — serves PDF bytes
│   ├── services/
│   │   ├── amadeus.go      # Amadeus API client + fallback data + destination highlights
│   │   ├── huggingface.go  # HuggingFace inference client
│   │   └── pdf.go          # PDF generation with gofpdf
│   ├── database/
│   │   └── db.go           # PostgreSQL schema + CRUD helpers
│   └── main.go
└── client/
    └── src/
        ├── pages/
        │   ├── Home.jsx    # Search form (with multi-city toggle)
        │   └── Results.jsx # Flight/hotel selection + confirm panel + PDF download
        └── services/
            └── api.js      # API calls
```

---

## How the budget works

Flight prices are **per person, round-trip**. The total cost shown in the confirm panel and PDF is:

```
total = (flight price × passengers) + (hotel per night × nights)
```

This was a deliberate design decision — the budget field represents what you're willing to spend across the whole group for flights plus accommodation.

---

## Multi-city trips

When you toggle "Multi-city return" on the search form, a third airport field appears. Enter the city you'll fly home from at the end of your trip. Internally, this triggers two separate one-way flight searches (outbound and return) which are combined into a single result set, same as a normal round-trip search. The PDF route section will show both legs clearly.

---

## Deploying

The backend has a `Dockerfile` and `railway.toml` — it's set up for Railway out of the box. Point `VITE_API_BASE_URL` in your frontend build to wherever the backend lands.

---

## Contributing

Issues and PRs welcome. The fallback route data in `amadeus.go` covers a couple dozen common routes — if yours is missing, it's a straightforward addition to the `knownRoutes` map. Same for destination highlights.

---

## License

MIT
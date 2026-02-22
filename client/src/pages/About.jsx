import "./About.css";

const FEATURES = [
  {
    icon: "🔍",
    title: "Smart Search",
    desc: "Enter your origin, destination, dates, and budget. We instantly surface the best flight and hotel combinations.",
  },
  {
    icon: "🤖",
    title: "AI Recommendations",
    desc: "Our AI analyses your options and budget to recommend the best-value flights and hotels, with clear reasoning.",
  },
  {
    icon: "📄",
    title: "Instant PDF",
    desc: "Finalise your selections and download a beautifully formatted itinerary PDF in one click — ready to share or print.",
  },
  {
    icon: "💰",
    title: "Budget Aware",
    desc: "Set your budget upfront. TripMind keeps total trip cost — flights plus hotel nights — front and centre.",
  },
  {
    icon: "🌍",
    title: "Multi-City Support",
    desc: "Search across popular European and Middle-Eastern routes with realistic pricing and airline data.",
  },
  {
    icon: "⚡",
    title: "Lightning Fast",
    desc: "Results appear in seconds. No waiting, no hidden fees, no endless redirects.",
  },
];

const API_ENDPOINTS = [
  { method: "POST", path: "/api/search", desc: "Search flights & hotels" },
  { method: "POST", path: "/api/generate", desc: "Generate itinerary PDF" },
  { method: "GET", path: "/api/download/:id", desc: "Download PDF by ID" },
  { method: "GET", path: "/api/health", desc: "Health check" },
];

export default function About() {
  return (
    <div className="about page-enter">
      {/* ── Hero ────────────────────────────────────────────── */}
      <section className="about__hero">
        <div className="about__hero-bg" aria-hidden="true" />
        <div className="about__hero-content">
          <span className="about__eyebrow animate-fadeUp">Open Source · MIT License</span>
          <h1 className="heading-display about__title animate-fadeUp delay-1">
            About TripMind
          </h1>
          <p className="about__sub animate-fadeUp delay-2">
            A smarter way to plan your next adventure — powered by AI
            and built for travellers who value their time.
          </p>
        </div>
      </section>

      {/* ── Features ────────────────────────────────────────── */}
      <section className="section about__section">
        <div className="about__section-head animate-fadeUp">
          <h2 className="heading-section">What it does</h2>
          <p className="about__section-sub">
            Everything you need to go from idea to itinerary — in seconds.
          </p>
        </div>

        <div className="about__features-grid">
          {FEATURES.map(({ icon, title, desc }, i) => (
            <div
              key={title}
              className={`about__feature-card animate-fadeUp delay-${i % 4 + 1}`}
            >
              <span className="about__feature-icon">{icon}</span>
              <h3 className="about__feature-title">{title}</h3>
              <p className="about__feature-desc">{desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* ── API Reference ───────────────────────────────────── */}
      <section className="section about__section" style={{ paddingTop: 0 }}>
        <div className="about__api-card animate-fadeUp delay-2">
          <div className="about__api-header">
            <div className="about__api-dot" />
            <h2 className="heading-section" style={{ fontSize: 20 }}>API Reference</h2>
          </div>
          <table className="about__api-table">
            <thead>
              <tr>
                <th>Method</th>
                <th>Endpoint</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              {API_ENDPOINTS.map(({ method, path, desc }) => (
                <tr key={path}>
                  <td>
                    <span className={`badge badge--method-${method.toLowerCase()}`}>
                      {method}
                    </span>
                  </td>
                  <td>
                    <code className="about__api-path">{path}</code>
                  </td>
                  <td className="about__api-desc">{desc}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
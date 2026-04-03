import "./About.css";

const HOW_IT_WORKS = [
  {
    step: "01",
    title: "Tell us where you're headed",
    desc: "Enter your departure city, destination, travel dates, and how much you'd like to spend. That's all we need to get started.",
  },
  {
    step: "02",
    title: "We find the best options",
    desc: "TripMind searches real flight and hotel availability and ranks them by value — so you spend less time comparing tabs.",
  },
  {
    step: "03",
    title: "Get a personalised recommendation",
    desc: "Our AI looks at your budget and options, then suggests the combination that makes the most sense for your trip.",
  },
  {
    step: "04",
    title: "Download your itinerary",
    desc: "Happy with the plan? Download a clean, ready-to-share PDF itinerary with all your trip details in one place.",
  },
];

const WHY_CARDS = [
  {
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
      </svg>
    ),
    title: "No hidden surprises",
    desc: "Prices you see include the full trip cost — flights plus hotel nights — so your budget stays yours.",
  },
  {
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="10"/>
        <polyline points="12 6 12 12 16 14"/>
      </svg>
    ),
    title: "Results in seconds",
    desc: "No endless loading bars. TripMind gives you real results fast, so you can focus on the exciting part — planning your trip.",
  },
  {
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
    ),
    title: "Honest recommendations",
    desc: "The AI explains its thinking — why it picked a certain flight or hotel — so you stay in control of your decisions.",
  },
  {
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
        <polyline points="14 2 14 8 20 8"/>
        <line x1="16" y1="13" x2="8" y2="13"/>
        <line x1="16" y1="17" x2="8" y2="17"/>
      </svg>
    ),
    title: "Trip PDF in one click",
    desc: "Your itinerary — flights, hotel, dates, costs — packaged into a clean PDF you can share, print, or keep for yourself.",
  },
  {
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="10"/>
        <line x1="2" y1="12" x2="22" y2="12"/>
        <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
      </svg>
    ),
    title: "Built for real routes",
    desc: "Popular routes from Central Asia to Europe and the Middle East — the trips real people actually want to take.",
  },
  {
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <line x1="12" y1="1" x2="12" y2="23"/>
        <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
      </svg>
    ),
    title: "Budget first, always",
    desc: "Set your budget at the start. TripMind never shows you options you can't afford — your limit is front and centre.",
  },
];

export default function About() {
  return (
    <div className="about page-enter">
      {/* ── Hero ── */}
      <section className="about__hero">
        <div className="about__hero-bg" aria-hidden="true">
          <div className="about__hero-glow" />
        </div>
        <div className="about__hero-content">
          <span className="about__eyebrow animate-fadeUp">Your travel companion</span>
          <h1 className="heading-display about__title animate-fadeUp delay-1">
            Planning a trip shouldn't<br />be a second job.
          </h1>
          <p className="about__sub animate-fadeUp delay-2">
            TripMind brings flights, hotels, and smart recommendations together
            in one place — so you can spend less time searching and more time
            looking forward to your trip.
          </p>
        </div>
      </section>

      {/* ── How it works ── */}
      <section className="section about__section">
        <div className="about__section-head animate-fadeUp">
          <h2 className="heading-section">How it works</h2>
          <p className="about__section-sub">Four simple steps from idea to itinerary.</p>
        </div>

        <div className="about__steps">
          {HOW_IT_WORKS.map(({ step, title, desc }, i) => (
            <div key={step} className={`about__step animate-fadeUp delay-${i % 3 + 1}`}>
              <div className="about__step-number">{step}</div>
              <div className="about__step-body">
                <h3 className="about__step-title">{title}</h3>
                <p className="about__step-desc">{desc}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* ── Why TripMind ── */}
      <section className="section about__section" style={{ paddingTop: 0 }}>
        <div className="about__section-head animate-fadeUp">
          <h2 className="heading-section">Why TripMind</h2>
          <p className="about__section-sub">Designed around the traveller, not the technology.</p>
        </div>

        <div className="about__features-grid">
          {WHY_CARDS.map(({ icon, title, desc }, i) => (
            <div key={title} className={`about__feature-card animate-fadeUp delay-${i % 4 + 1}`}>
              <span className="about__feature-icon">{icon}</span>
              <h3 className="about__feature-title">{title}</h3>
              <p className="about__feature-desc">{desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* ── Closing note ── */}
      <section className="about__closing animate-fadeUp">
        <div className="about__closing-inner">
          <p className="about__closing-text">
            TripMind was built as a final-year project at Singapore University of Management in Tashkent,
            with a simple goal: make travel planning feel effortless.
          </p>
        </div>
      </section>
    </div>
  );
}

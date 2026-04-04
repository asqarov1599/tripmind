import {
  Shield,
  Clock,
  MessageSquare,
  FileText,
  Globe,
  DollarSign,
} from "lucide-react";
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
    icon: <Shield size={22} />,
    title: "No hidden surprises",
    desc: "Prices you see include the full trip cost — flights plus hotel nights — so your budget stays yours.",
  },
  {
    icon: <Clock size={22} />,
    title: "Results in seconds",
    desc: "No endless loading bars. TripMind gives you real results fast, so you can focus on the exciting part — planning your trip.",
  },
  {
    icon: <MessageSquare size={22} />,
    title: "Honest recommendations",
    desc: "The AI explains its thinking — why it picked a certain flight or hotel — so you stay in control of your decisions.",
  },
  {
    icon: <FileText size={22} />,
    title: "Trip PDF in one click",
    desc: "Your itinerary — flights, hotel, dates, costs — packaged into a clean PDF you can share, print, or keep for yourself.",
  },
  {
    icon: <Globe size={22} />,
    title: "Built for real routes",
    desc: "Popular routes from Central Asia to Europe and the Middle East — the trips real people actually want to take.",
  },
  {
    icon: <DollarSign size={22} />,
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
            TripMind was built as a final-year project at Management Development Institute of Singapore in Tashkent,
            with a simple goal: make travel planning feel effortless.
          </p>
        </div>
      </section>
    </div>
  );
}

import "./Navbar.css";

export default function Navbar({ currentPage, onNavigate }) {
  return (
    <header className="navbar">
      <div className="navbar__inner">
        <button className="navbar__logo" onClick={() => onNavigate("home")}>
          <span className="navbar__logo-mark">T</span>
          <span className="navbar__logo-text">
            trip<strong>mind</strong>
          </span>
        </button>

        <nav className="navbar__links">
          <button
            className={`navbar__link ${currentPage === "home" || currentPage === "results" ? "navbar__link--active" : ""}`}
            onClick={() => onNavigate("home")}
          >
            Search
          </button>
          <button
            className={`navbar__link ${currentPage === "about" ? "navbar__link--active" : ""}`}
            onClick={() => onNavigate("about")}
          >
            About
          </button>
        </nav>
      </div>
    </header>
  );
}
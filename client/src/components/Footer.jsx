import "./Footer.css";

export default function Footer() {
  return (
    <footer className="footer">
      <div className="footer__inner">
        <span className="footer__brand">tripmind</span>
        <span className="footer__divider">·</span>
        <span className="footer__text">AI-Powered Travel Planning</span>
        <span className="footer__divider">·</span>
        <span className="footer__text">
          Backend{" "}
          <code className="footer__code">localhost:8080</code>
        </span>
      </div>
    </footer>
  );
}
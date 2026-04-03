import "./Footer.css";

export default function Footer() {
  return (
    <footer className="footer">
      <div className="footer__inner">
        <span className="footer__brand">tripmind</span>
        <span className="footer__divider">·</span>
        <span className="footer__text">Travel smarter, worry less</span>
        <span className="footer__divider">·</span>
        <span className="footer__text footer__text--muted">Built in Tashkent</span>
      </div>
    </footer>
  );
}

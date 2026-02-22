import { useState, useCallback } from "react";

import "./styles/global.css";
import "./styles/utilities.css";
import "./styles/animations.css";

import Navbar from "./components/Navbar";
import Footer from "./components/Footer";

import Home from "./pages/Home";
import Results from "./pages/Results";
import About from "./pages/About";

export default function App() {
  const [page, setPage] = useState("home");
  const [results, setResults] = useState(null);
  const [searchForm, setSearchForm] = useState(null);

  const handleResults = useCallback((data, form) => {
    setResults(data);
    setSearchForm(form);
    setPage("results");
    window.scrollTo({ top: 0, behavior: "smooth" });
  }, []);

  const navigate = useCallback((dest) => {
    setPage(dest);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }, []);

  return (
    <>
      <Navbar currentPage={page} onNavigate={navigate} />

      <main>
        {page === "home" && <Home onResults={handleResults} />}
        {page === "results" && results && (
          <Results
            data={results}
            searchForm={searchForm}
            onBack={() => navigate("home")}
          />
        )}
        {page === "about" && <About />}
      </main>

      <Footer />
    </>
  );
}
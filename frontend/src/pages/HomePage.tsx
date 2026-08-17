import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

interface SearchResult {
  symbol: string;
  name: string;
  exchange: string;
}

interface FmpReferenceData {
  symbol: string;
  provider: string;
  available: boolean;
  dcfValue?: number;
  leveredDcfValue?: number;
  asOfDate?: string;
  message?: string;
}

export default function HomePage() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [recentTickers, setRecentTickers] = useState<string[]>([]);
  const [status, setStatus] = useState("");
  const [provider, setProvider] = useState("");
  const [providerMessage, setProviderMessage] = useState("");
  const [selectedReferenceSymbol, setSelectedReferenceSymbol] = useState("");
  const [fmpReference, setFmpReference] = useState<FmpReferenceData | null>(null);
  const [loadingFmpReference, setLoadingFmpReference] = useState(false);
  const [fmpReferenceStatus, setFmpReferenceStatus] = useState("");

  useEffect(() => {
    try {
      const raw = localStorage.getItem("recent-tickers");
      if (!raw) return;
      const parsed = JSON.parse(raw) as string[];
      if (Array.isArray(parsed)) setRecentTickers(parsed.slice(0, 6));
    } catch {
      setRecentTickers([]);
    }
  }, []);

  async function onSearch() {
    const trimmed = query.trim();
    if (!trimmed) {
      setResults([]);
      setStatus("");
      setProvider("");
      setProviderMessage("");
      return;
    }
    setLoading(true);
    const response = await fetch(`/api/v1/tickers/search?q=${encodeURIComponent(trimmed)}`).catch(() => null);
    if (!response || !response.ok) {
      setResults([]);
      setStatus("Search failed. Please try again.");
      setProvider("");
      setProviderMessage("");
      setLoading(false);
      return;
    }
    const payload = (await response.json()) as {
      results?: SearchResult[];
      provider?: string;
      providerMessage?: string;
    };
    const fetched = payload.results ?? [];
    setResults(fetched);
    setStatus(fetched.length === 0 ? "No tickers found." : "");
    setProvider(payload.provider ?? "");
    setProviderMessage(payload.providerMessage ?? "");
    const firstSymbol = fetched[0]?.symbol ?? "";
    setSelectedReferenceSymbol(firstSymbol);
    if (!firstSymbol) {
      setFmpReference(null);
      setFmpReferenceStatus("");
    }
    setLoading(false);
  }

  useEffect(() => {
    const trimmed = query.trim();
    if (!trimmed) {
      setResults([]);
      setStatus("");
      return;
    }
    const timeout = setTimeout(() => {
      onSearch().catch(() => setStatus("Search failed. Please try again."));
    }, 300);
    return () => clearTimeout(timeout);
  }, [query]);

  function rememberTicker(symbol: string) {
    const next = [symbol, ...recentTickers.filter((item) => item !== symbol)].slice(0, 6);
    setRecentTickers(next);
    localStorage.setItem("recent-tickers", JSON.stringify(next));
  }

  useEffect(() => {
    if (!selectedReferenceSymbol) return;
    setLoadingFmpReference(true);
    setFmpReferenceStatus("");
    fetch(`/api/v1/tickers/${selectedReferenceSymbol}/fmp-reference`)
      .then(async (response) => {
        if (!response.ok) throw new Error("Unable to fetch FMP reference values.");
        const payload = (await response.json()) as FmpReferenceData;
        setFmpReference(payload);
      })
      .catch(() => {
        setFmpReference(null);
        setFmpReferenceStatus("Unable to load FMP reference values right now.");
      })
      .finally(() => setLoadingFmpReference(false));
  }, [selectedReferenceSymbol]);

  return (
    <main>
      <div className="card">
        <h1>Stock Advisor DCF Prototype</h1>
        <p>Search a ticker to open the MVP DCF workspace.</p>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <input
            placeholder="Try NVDA or AAPL"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            style={{ width: 280, padding: "8px 10px", borderRadius: 8, border: "1px solid #ccc" }}
          />
          <button className="button" onClick={onSearch} disabled={loading}>
            {loading ? "Searching..." : "Search"}
          </button>
        </div>
        <p style={{ color: "#666", fontSize: 12, marginTop: 8 }}>Search updates automatically while typing.</p>
      </div>

      <div style={{ marginTop: 16 }} className="card">
        <h2>Recent Tickers</h2>
        {recentTickers.length === 0 ? <p>No recent tickers yet.</p> : null}
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          {recentTickers.map((symbol) => (
            <Link
              key={symbol}
              to={`/stock/${symbol}/dcf`}
              className="button"
              onClick={() => rememberTicker(symbol)}
            >
              {symbol}
            </Link>
          ))}
        </div>
      </div>

      <div style={{ marginTop: 16 }} className="card">
        <h2>Results</h2>
        {provider ? <p style={{ color: "#666", fontSize: 12 }}>Data source: {provider}</p> : null}
        {provider === "seed-fallback" ? (
          <p style={{ color: "#9a6700", fontSize: 12 }}>Live provider unavailable, showing fallback seed data.</p>
        ) : null}
        {providerMessage ? <p style={{ color: "#9a6700", fontSize: 12 }}>{providerMessage}</p> : null}
        {results.length === 0 ? <p>{status || "No results yet."}</p> : null}
        <ul>
          {results.map((result) => (
            <li key={result.symbol} style={{ marginBottom: 8 }}>
              <Link to={`/stock/${result.symbol}/dcf`} onClick={() => rememberTicker(result.symbol)}>
                {result.symbol} - {result.name} ({result.exchange})
              </Link>
              <button
                type="button"
                className="button"
                style={{ marginLeft: 8, padding: "3px 8px", fontSize: 12 }}
                onClick={() => setSelectedReferenceSymbol(result.symbol)}
              >
                Reference
              </button>
            </li>
          ))}
        </ul>
        {selectedReferenceSymbol ? (
          <div style={{ marginTop: 12, borderTop: "1px solid #eee", paddingTop: 12 }}>
            <h3 style={{ margin: "0 0 8px", fontSize: 14 }}>
              FMP Reference Values ({selectedReferenceSymbol})
            </h3>
            <p style={{ color: "#666", fontSize: 12, marginTop: 0 }}>
              Informational reference from Financial Modeling Prep endpoints.
            </p>
            {loadingFmpReference ? <p style={{ fontSize: 12 }}>Loading reference values...</p> : null}
            {!loadingFmpReference && fmpReference?.available ? (
              <>
                <p style={{ margin: "4px 0" }}>DCF value: {fmpReference.dcfValue ?? "-"}</p>
                <p style={{ margin: "4px 0" }}>DCF levered: {fmpReference.leveredDcfValue ?? "-"}</p>
                {fmpReference.asOfDate ? (
                  <p style={{ margin: "4px 0", color: "#666", fontSize: 12 }}>As of: {fmpReference.asOfDate}</p>
                ) : null}
              </>
            ) : null}
            {!loadingFmpReference && fmpReference && !fmpReference.available ? (
              <p style={{ color: "#9a6700", fontSize: 12 }}>{fmpReference.message ?? "FMP reference unavailable."}</p>
            ) : null}
            {fmpReferenceStatus ? <p style={{ color: "#9a6700", fontSize: 12 }}>{fmpReferenceStatus}</p> : null}
          </div>
        ) : null}
      </div>
    </main>
  );
}

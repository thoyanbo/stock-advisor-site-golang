import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  deriveGrowthRateFromHistoricalRows,
  evaluateGrowthPlausibility,
  type BasisType,
  type GrowthMetric,
  type HistoricalGrowthRow,
  type ScenarioRecord,
} from "../lib/growth";

interface SummaryResponse {
  symbol: string;
  name: string;
  exchange: string;
  currency: string;
  price: number;
  valuationBases: { eps: number; fcf: number; dividend: number };
  tangibleBookValuePerShare: number;
  provider?: string;
  providerMessage?: string;
}

interface DCFResult {
  fairValue: number;
  marginOfSafetyPct: number;
  presentValueGrowthStage: number;
  presentValueTerminalStage: number;
}

interface ReverseDCFResult {
  impliedGrowthRatePct: number;
  modelPriceAtImpliedGrowth: number;
  converged: boolean;
  iterations: number;
}

interface HistoricalGrowthResponse {
  rows: HistoricalGrowthRow[];
  historicalSeries?: {
    eps: { years: number[]; values: number[] };
    revenue: { years: number[]; values: number[] };
    operatingIncome: { years: number[]; values: number[] };
    ebitda: { years: number[]; values: number[] };
    fcf: { years: number[]; values: number[] };
    dividend: { years: number[]; values: number[] };
    bookValue: { years: number[]; values: number[] };
    price: { years: number[]; values: number[] };
  };
  dataQuality?: { score: number; coveragePct: number; level: "high" | "medium" | "low"; message: string };
  provider?: string;
  providerMessage?: string;
  historicalDiagnostics?: {
    supplementStatus: "none" | "finnhub-applied" | "finnhub-no-fill" | "finnhub-failed";
    filledPoints?: number;
  };
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

const METRIC_LABELS: Record<GrowthMetric, string> = {
  eps: "EPS",
  revenue: "Revenue",
  operatingIncome: "Operating Income",
  ebitda: "EBITDA",
  fcf: "Free Cash Flow",
  dividend: "Dividends",
  bookValue: "Book Value",
  price: "Price",
};

const DEFAULT_ASSUMPTIONS = {
  basisType: "eps" as BasisType,
  discountRatePct: 11,
  growthYears: 10,
  growthRatePct: 20,
  terminalYears: 10,
  terminalRatePct: 4,
  addTangibleBook: true,
};

export default function DCFPage() {
  const { ticker: rawTicker } = useParams<{ ticker: string }>();
  const ticker = (rawTicker ?? "").toUpperCase();

  const [summary, setSummary] = useState<SummaryResponse | null>(null);
  const [error, setError] = useState("");
  const [loadingSummary, setLoadingSummary] = useState(true);
  const [loadingValuation, setLoadingValuation] = useState(false);
  const [result, setResult] = useState<DCFResult | null>(null);
  const [reverseResult, setReverseResult] = useState<ReverseDCFResult | null>(null);
  const [historicalRows, setHistoricalRows] = useState<HistoricalGrowthRow[]>([]);
  const [growthMessage, setGrowthMessage] = useState("");
  const [scenarios, setScenarios] = useState<ScenarioRecord[]>([]);
  const [scenarioName, setScenarioName] = useState("Base Case");
  const [selectedScenarioId, setSelectedScenarioId] = useState("");
  const [scenarioStatus, setScenarioStatus] = useState("");
  const [scenarioBusy, setScenarioBusy] = useState(false);
  const [assumptionsHydrated, setAssumptionsHydrated] = useState(false);
  const [hasStoredGrowthRate, setHasStoredGrowthRate] = useState(false);
  const [growthRateTouched, setGrowthRateTouched] = useState(false);
  const [summaryProvider, setSummaryProvider] = useState("");
  const [historicalProvider, setHistoricalProvider] = useState("");
  const [summaryProviderMessage, setSummaryProviderMessage] = useState("");
  const [historicalProviderMessage, setHistoricalProviderMessage] = useState("");
  const [historicalDiagnostics, setHistoricalDiagnostics] =
    useState<HistoricalGrowthResponse["historicalDiagnostics"]>(undefined);
  const [historicalSeries, setHistoricalSeries] =
    useState<HistoricalGrowthResponse["historicalSeries"]>(undefined);
  const [showGrowthDebug, setShowGrowthDebug] = useState(false);
  const [dataQuality, setDataQuality] =
    useState<HistoricalGrowthResponse["dataQuality"]>(undefined);
  const [fmpReference, setFmpReference] = useState<FmpReferenceData | null>(null);
  const [loadingFmpReference, setLoadingFmpReference] = useState(false);
  const [fmpReferenceStatus, setFmpReferenceStatus] = useState("");

  const [basisType, setBasisType] = useState<BasisType>(DEFAULT_ASSUMPTIONS.basisType);
  const [discountRatePct, setDiscountRatePct] = useState(DEFAULT_ASSUMPTIONS.discountRatePct);
  const [growthYears, setGrowthYears] = useState(DEFAULT_ASSUMPTIONS.growthYears);
  const [growthRatePct, setGrowthRatePct] = useState(DEFAULT_ASSUMPTIONS.growthRatePct);
  const [terminalYears, setTerminalYears] = useState(DEFAULT_ASSUMPTIONS.terminalYears);
  const [terminalRatePct, setTerminalRatePct] = useState(DEFAULT_ASSUMPTIONS.terminalRatePct);
  const [addTangibleBook, setAddTangibleBook] = useState(DEFAULT_ASSUMPTIONS.addTangibleBook);

  useEffect(() => {
    if (!ticker) return;
    async function loadData() {
      setLoadingSummary(true);
      setError("");
      const [summaryResp, growthResp, scenariosResp] = await Promise.all([
        fetch(`/api/v1/tickers/${ticker}/summary`),
        fetch(`/api/v1/tickers/${ticker}/historical-growth`),
        fetch(`/api/v1/tickers/${ticker}/scenarios`),
      ]);
      if (!summaryResp.ok) {
        setSummary(null);
        setError("Unable to load ticker summary.");
        setLoadingSummary(false);
        return;
      }
      const sp = (await summaryResp.json()) as SummaryResponse;
      setSummary(sp);
      setSummaryProvider(sp.provider ?? "");
      setSummaryProviderMessage(sp.providerMessage ?? "");
      if (growthResp.ok) {
        const gp = (await growthResp.json()) as HistoricalGrowthResponse;
        setHistoricalRows(gp.rows ?? []);
        setHistoricalSeries(gp.historicalSeries);
        setDataQuality(gp.dataQuality);
        setHistoricalProvider(gp.provider ?? "");
        setHistoricalProviderMessage(gp.providerMessage ?? "");
        setHistoricalDiagnostics(gp.historicalDiagnostics);
      } else {
        setHistoricalRows([]);
        setHistoricalSeries(undefined);
        setDataQuality(undefined);
        setHistoricalProvider("");
        setHistoricalProviderMessage("");
        setHistoricalDiagnostics(undefined);
      }
      if (scenariosResp.ok) {
        const sp2 = (await scenariosResp.json()) as { scenarios: ScenarioRecord[] };
        setScenarios(sp2.scenarios ?? []);
      } else {
        setScenarios([]);
      }
      setLoadingSummary(false);
    }
    loadData().catch(() => {
      setError("Unable to load ticker summary.");
      setLoadingSummary(false);
    });
  }, [ticker]);

  useEffect(() => {
    if (!ticker) return;
    setLoadingFmpReference(true);
    setFmpReferenceStatus("");
    fetch(`/api/v1/tickers/${ticker}/fmp-reference`)
      .then(async (r) => {
        if (!r.ok) throw new Error();
        setFmpReference((await r.json()) as FmpReferenceData);
      })
      .catch(() => {
        setFmpReference(null);
        setFmpReferenceStatus("Unable to load FMP reference values.");
      })
      .finally(() => setLoadingFmpReference(false));
  }, [ticker]);

  const baseValuePerShare = useMemo(() => {
    if (!summary) return 0;
    return summary.valuationBases[basisType];
  }, [summary, basisType]);

  const assumptionsError = useMemo(() => {
    if (discountRatePct <= 0) return "Discount rate must be greater than 0.";
    if (growthYears < 1 || terminalYears < 1) return "Growth and terminal years must be at least 1.";
    if (terminalRatePct >= discountRatePct) return "Terminal growth rate must stay below discount rate.";
    return "";
  }, [discountRatePct, growthYears, terminalYears, terminalRatePct]);

  useEffect(() => {
    if (!ticker) return;
    try {
      const raw = localStorage.getItem("recent-tickers");
      const parsed = raw ? (JSON.parse(raw) as string[]) : [];
      const next = [ticker, ...parsed.filter((s) => s !== ticker)].slice(0, 6);
      localStorage.setItem("recent-tickers", JSON.stringify(next));
    } catch { /* best-effort */ }
  }, [ticker]);

  useEffect(() => {
    if (!ticker) return;
    setAssumptionsHydrated(false);
    setHasStoredGrowthRate(false);
    setGrowthRateTouched(false);
    const key = `dcf-assumptions-${ticker}`;
    try {
      const raw = localStorage.getItem(key);
      if (raw) {
        const p = JSON.parse(raw) as Partial<typeof DEFAULT_ASSUMPTIONS>;
        if (p.basisType) setBasisType(p.basisType);
        if (typeof p.discountRatePct === "number") setDiscountRatePct(p.discountRatePct);
        if (typeof p.growthYears === "number") setGrowthYears(p.growthYears);
        if (typeof p.growthRatePct === "number") {
          setGrowthRatePct(p.growthRatePct);
          setHasStoredGrowthRate(true);
          setGrowthRateTouched(true);
        }
        if (typeof p.terminalYears === "number") setTerminalYears(p.terminalYears);
        if (typeof p.terminalRatePct === "number") setTerminalRatePct(p.terminalRatePct);
        if (typeof p.addTangibleBook === "boolean") setAddTangibleBook(p.addTangibleBook);
      }
    } catch { /* ignore */ } finally {
      setAssumptionsHydrated(true);
    }
  }, [ticker]);

  const suggestedGrowthRatePct = useMemo(
    () => deriveGrowthRateFromHistoricalRows(basisType, historicalRows, DEFAULT_ASSUMPTIONS.growthRatePct),
    [basisType, historicalRows]
  );

  const historicalDiagnosticsMessage = useMemo(() => {
    if (!historicalDiagnostics || historicalDiagnostics.supplementStatus === "none") return "";
    if (historicalDiagnostics.supplementStatus === "finnhub-applied")
      return `Historical supplement: Finnhub applied (${historicalDiagnostics.filledPoints ?? 0} points backfilled).`;
    if (historicalDiagnostics.supplementStatus === "finnhub-no-fill")
      return "Historical supplement: Finnhub checked, but no additional annual points were available.";
    return "Historical supplement: Finnhub request failed; showing FMP-only historical fields.";
  }, [historicalDiagnostics]);

  const historicalSourceBadge = useMemo(() => {
    if (historicalProvider === "seed" || historicalProvider === "seed-fallback")
      return { label: "Seed", color: "#666", background: "#f3f4f6" };
    if (historicalDiagnostics?.supplementStatus === "finnhub-applied")
      return { label: "FMP+Finnhub", color: "#1a7f37", background: "#eaf8ee" };
    if (historicalDiagnostics?.supplementStatus === "finnhub-failed")
      return { label: "FMP-only", color: "#b42318", background: "#fef3f2" };
    if (historicalProvider === "fmp") return { label: "FMP-only", color: "#155eef", background: "#eef4ff" };
    return null;
  }, [historicalProvider, historicalDiagnostics]);

  useEffect(() => {
    if (!assumptionsHydrated || hasStoredGrowthRate || growthRateTouched) return;
    setGrowthRatePct(suggestedGrowthRatePct);
  }, [assumptionsHydrated, hasStoredGrowthRate, growthRateTouched, suggestedGrowthRatePct]);

  useEffect(() => {
    if (!ticker || !assumptionsHydrated) return;
    const key = `dcf-assumptions-${ticker}`;
    localStorage.setItem(key, JSON.stringify({
      basisType, discountRatePct, growthYears, growthRatePct, terminalYears, terminalRatePct, addTangibleBook,
    }));
  }, [ticker, assumptionsHydrated, basisType, discountRatePct, growthYears, growthRatePct, terminalYears, terminalRatePct, addTangibleBook]);

  function resetAssumptions() {
    setBasisType(DEFAULT_ASSUMPTIONS.basisType);
    setDiscountRatePct(DEFAULT_ASSUMPTIONS.discountRatePct);
    setGrowthYears(DEFAULT_ASSUMPTIONS.growthYears);
    setGrowthRatePct(DEFAULT_ASSUMPTIONS.growthRatePct);
    setTerminalYears(DEFAULT_ASSUMPTIONS.terminalYears);
    setTerminalRatePct(DEFAULT_ASSUMPTIONS.terminalRatePct);
    setAddTangibleBook(DEFAULT_ASSUMPTIONS.addTangibleBook);
    setScenarioStatus("Assumptions reset to defaults.");
  }

  function buildScenarioPayload() {
    return {
      name: scenarioName.trim() || "Untitled Scenario",
      basisType, discountRatePct, growthYears, growthRatePct, terminalYears, terminalRatePct, addTangibleBook,
    };
  }

  function applyScenario(s: ScenarioRecord) {
    setBasisType(s.basisType);
    setDiscountRatePct(s.discountRatePct);
    setGrowthYears(s.growthYears);
    setGrowthRatePct(s.growthRatePct);
    setTerminalYears(s.terminalYears);
    setTerminalRatePct(s.terminalRatePct);
    setAddTangibleBook(s.addTangibleBook);
    setScenarioName(s.name);
    setSelectedScenarioId(s.id);
    setScenarioStatus(`Loaded "${s.name}".`);
  }

  async function refreshScenarios() {
    if (!ticker) return;
    const r = await fetch(`/api/v1/tickers/${ticker}/scenarios`);
    if (!r.ok) return;
    const p = (await r.json()) as { scenarios: ScenarioRecord[] };
    setScenarios(p.scenarios ?? []);
  }

  async function saveScenario() {
    if (!ticker) return;
    setScenarioBusy(true);
    setScenarioStatus("");
    const r = await fetch(`/api/v1/tickers/${ticker}/scenarios`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(buildScenarioPayload()),
    });
    const p = (await r.json()) as { scenario?: ScenarioRecord; error?: string };
    if (!r.ok || !p.scenario) {
      setScenarioStatus(p.error ?? "Unable to save scenario.");
      setScenarioBusy(false);
      return;
    }
    await refreshScenarios();
    setSelectedScenarioId(p.scenario.id);
    setScenarioStatus(`Saved "${p.scenario.name}".`);
    setScenarioBusy(false);
  }

  async function updateSelectedScenario() {
    if (!ticker || !selectedScenarioId) { setScenarioStatus("Select a scenario to update."); return; }
    setScenarioBusy(true);
    setScenarioStatus("");
    const r = await fetch(`/api/v1/tickers/${ticker}/scenarios/${selectedScenarioId}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(buildScenarioPayload()),
    });
    const p = (await r.json()) as { scenario?: ScenarioRecord; error?: string };
    if (!r.ok || !p.scenario) { setScenarioStatus(p.error ?? "Unable to update scenario."); setScenarioBusy(false); return; }
    await refreshScenarios();
    setScenarioStatus(`Updated "${p.scenario.name}".`);
    setScenarioBusy(false);
  }

  async function deleteSelectedScenario() {
    if (!ticker || !selectedScenarioId) { setScenarioStatus("Select a scenario to delete."); return; }
    if (!window.confirm("Delete selected scenario?")) return;
    setScenarioBusy(true);
    setScenarioStatus("");
    const r = await fetch(`/api/v1/tickers/${ticker}/scenarios/${selectedScenarioId}`, { method: "DELETE" });
    if (!r.ok) {
      const p = (await r.json()) as { error?: string };
      setScenarioStatus(p.error ?? "Unable to delete scenario.");
      setScenarioBusy(false);
      return;
    }
    await refreshScenarios();
    setSelectedScenarioId("");
    setScenarioStatus("Scenario deleted.");
    setScenarioBusy(false);
  }

  async function runValuation() {
    if (!summary) return;
    if (assumptionsError) { setError(assumptionsError); return; }
    setLoadingValuation(true);
    setError("");
    const body = {
      basisType,
      currentPrice: summary.price,
      baseValuePerShare,
      discountRatePct, growthYears, growthRatePct, terminalYears, terminalRatePct,
      addTangibleBook,
      tangibleBookValuePerShare: summary.tangibleBookValuePerShare,
    };
    const [dcfR, revR] = await Promise.all([
      fetch("/api/v1/valuation/dcf", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
      fetch("/api/v1/valuation/reverse-dcf", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          currentPrice: summary.price, baseValuePerShare, discountRatePct, growthYears, terminalYears, terminalRatePct, addTangibleBook,
          tangibleBookValuePerShare: summary.tangibleBookValuePerShare,
        }),
      }),
    ]);
    const dcfP = (await dcfR.json()) as { result?: DCFResult; error?: string };
    const revP = (await revR.json()) as { result?: ReverseDCFResult; error?: string };
    if (!dcfR.ok || !dcfP.result) { setError(dcfP.error ?? "Valuation failed."); setLoadingValuation(false); return; }
    if (!revR.ok || !revP.result) { setError(revP.error ?? "Reverse DCF failed."); setLoadingValuation(false); return; }
    setResult(dcfP.result);
    setReverseResult(revP.result);
    setGrowthMessage(evaluateGrowthPlausibility(revP.result.impliedGrowthRatePct, historicalRows).message);
    setLoadingValuation(false);
  }

  return (
    <main>
      <p><Link to="/">Back to search</Link></p>
      <h1>DCF Workspace: {ticker || "-"}</h1>

      {loadingSummary ? <p>Loading summary...</p> : null}
      {summary ? (
        <div className="card" style={{ marginBottom: 16 }}>
          <strong>{summary.name} ({summary.symbol}) - {summary.exchange}</strong>
          <p>Current price: {summary.currency} {summary.price}</p>
          <p>Base value ({basisType}): {baseValuePerShare.toFixed(2)}</p>
          {summaryProvider ? <p style={{ color: "#666", fontSize: 12 }}>Summary source: {summaryProvider}</p> : null}
          {historicalProvider ? (
            <p style={{ color: "#666", fontSize: 12 }}>
              Historical source: {historicalProvider}
              {historicalSourceBadge ? (
                <span
                  title={historicalDiagnosticsMessage || `Historical valuation source badge: ${historicalSourceBadge.label}`}
                  style={{
                    marginLeft: 8, padding: "2px 6px", borderRadius: 999, fontSize: 11, fontWeight: 600,
                    color: historicalSourceBadge.color, background: historicalSourceBadge.background,
                  }}
                >
                  {historicalSourceBadge.label}
                </span>
              ) : null}
            </p>
          ) : null}
          {dataQuality ? (
            <p style={{
              color: dataQuality.level === "high" ? "#1a7f37" : dataQuality.level === "medium" ? "#9a6700" : "#b42318",
              fontSize: 12,
            }}>
              Data quality: {dataQuality.level.toUpperCase()} ({dataQuality.coveragePct}% coverage) - {dataQuality.message}
            </p>
          ) : null}
          {(summaryProvider === "seed-fallback" || historicalProvider === "seed-fallback") && (
            <p style={{ color: "#9a6700", fontSize: 12 }}>Live provider fallback active for part of this ticker data.</p>
          )}
          {summaryProviderMessage ? <p style={{ color: "#9a6700", fontSize: 12 }}>{summaryProviderMessage}</p> : null}
          {historicalProviderMessage ? <p style={{ color: "#9a6700", fontSize: 12 }}>{historicalProviderMessage}</p> : null}
          {historicalDiagnosticsMessage ? <p style={{ color: "#666", fontSize: 12 }}>{historicalDiagnosticsMessage}</p> : null}
          <div style={{ marginTop: 10, paddingTop: 10, borderTop: "1px solid #eee" }}>
            <p style={{ margin: 0, fontWeight: 600, fontSize: 13 }}>FMP Reference Values (informational)</p>
            {loadingFmpReference ? <p style={{ margin: "6px 0 0", fontSize: 12 }}>Loading...</p> : null}
            {!loadingFmpReference && fmpReference?.available ? (
              <>
                <p style={{ margin: "6px 0 0", fontSize: 12 }}>DCF value: {fmpReference.dcfValue ?? "-"}</p>
                <p style={{ margin: "4px 0 0", fontSize: 12 }}>DCF levered: {fmpReference.leveredDcfValue ?? "-"}</p>
                {fmpReference.asOfDate ? <p style={{ margin: "4px 0 0", color: "#666", fontSize: 12 }}>As of: {fmpReference.asOfDate}</p> : null}
              </>
            ) : null}
            {!loadingFmpReference && fmpReference && !fmpReference.available ? (
              <p style={{ margin: "6px 0 0", color: "#9a6700", fontSize: 12 }}>{fmpReference.message ?? "FMP reference unavailable."}</p>
            ) : null}
            {fmpReferenceStatus ? <p style={{ margin: "6px 0 0", color: "#9a6700", fontSize: 12 }}>{fmpReferenceStatus}</p> : null}
          </div>
        </div>
      ) : null}

      <div className="grid grid-3">
        <div className="card">
          <h2>Assumptions</h2>
          <div className="field">
            <label>Basis</label>
            <select value={basisType} onChange={(e) => setBasisType(e.target.value as BasisType)}>
              <option value="eps">EPS</option>
              <option value="fcf">FCF</option>
              <option value="dividend">Dividend</option>
            </select>
          </div>
          <div className="field">
            <label>Discount Rate (%)</label>
            <input type="number" value={discountRatePct} onChange={(e) => setDiscountRatePct(Number(e.target.value))} />
          </div>
          <div className="field">
            <label>Growth Years</label>
            <input type="number" value={growthYears} onChange={(e) => setGrowthYears(Number(e.target.value))} />
          </div>
          <div className="field">
            <label>Growth Rate (%)</label>
            <input type="number" value={growthRatePct} onChange={(e) => { setGrowthRateTouched(true); setGrowthRatePct(Number(e.target.value)); }} />
            <p style={{ margin: "6px 0 0", color: "#666", fontSize: 12 }}>
              Auto suggestion: {suggestedGrowthRatePct}% (prefers 10Y growth, then 5Y/3Y/1Y, clamped to 5%-20%).
            </p>
          </div>
          <div className="field">
            <label>Terminal Years</label>
            <input type="number" value={terminalYears} onChange={(e) => setTerminalYears(Number(e.target.value))} />
          </div>
          <div className="field">
            <label>Terminal Rate (%)</label>
            <input type="number" value={terminalRatePct} onChange={(e) => setTerminalRatePct(Number(e.target.value))} />
          </div>
          <label style={{ display: "flex", gap: 8, marginTop: 10 }}>
            <input type="checkbox" checked={addTangibleBook} onChange={(e) => setAddTangibleBook(e.target.checked)} />
            Add tangible book value
          </label>
          <div style={{ display: "flex", gap: 8, marginTop: 12, flexWrap: "wrap" }}>
            <button className="button" onClick={runValuation} disabled={loadingValuation || !summary || Boolean(assumptionsError)}>
              {loadingValuation ? "Calculating..." : "Run DCF"}
            </button>
            <button className="button" onClick={resetAssumptions} type="button">Reset Defaults</button>
          </div>
          {assumptionsError ? <p style={{ color: "crimson", marginTop: 8 }}>{assumptionsError}</p> : null}

          <hr style={{ margin: "16px 0" }} />
          <h3>Scenarios</h3>
          <div className="field">
            <label>Scenario Name</label>
            <input value={scenarioName} onChange={(e) => setScenarioName(e.target.value)} />
          </div>
          <div style={{ display: "flex", gap: 8, marginTop: 10, flexWrap: "wrap" }}>
            <button className="button" onClick={saveScenario} disabled={scenarioBusy}>Save New</button>
            <button className="button" onClick={updateSelectedScenario} disabled={scenarioBusy || !selectedScenarioId}>Update Selected</button>
            <button className="button" onClick={deleteSelectedScenario} disabled={scenarioBusy || !selectedScenarioId}>Delete Selected</button>
          </div>
          {scenarioStatus ? <p style={{ marginTop: 8 }}>{scenarioStatus}</p> : null}
          <div style={{ marginTop: 10 }}>
            {scenarios.length === 0 ? <p>No saved scenarios yet.</p> : null}
            {scenarios.map((s) => (
              <div key={s.id} style={{ display: "flex", justifyContent: "space-between", alignItems: "center", border: "1px solid #ddd", borderRadius: 8, padding: "6px 8px", marginBottom: 6 }}>
                <div>
                  <strong>{s.name}</strong>
                  <div style={{ fontSize: 12, color: "#666" }}>{new Date(s.updatedAt).toLocaleString()}</div>
                </div>
                <div style={{ display: "flex", gap: 6 }}>
                  <button className="button" onClick={() => applyScenario(s)} disabled={scenarioBusy} style={{ padding: "4px 8px" }}>Load</button>
                  <button className="button" onClick={() => { setSelectedScenarioId(s.id); setScenarioName(s.name); setScenarioStatus(`Selected "${s.name}".`); }} disabled={scenarioBusy} style={{ padding: "4px 8px" }}>Select</button>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <h2>Output</h2>
          {!result ? <p>No valuation run yet.</p> : null}
          {result ? (
            <>
              <p><strong>Fair Value:</strong> {summary?.currency} {result.fairValue}</p>
              <p><strong>Margin of Safety:</strong> {result.marginOfSafetyPct}%</p>
              <p>PV (Growth Stage): {result.presentValueGrowthStage} | PV (Terminal Stage): {result.presentValueTerminalStage}</p>
            </>
          ) : null}
        </div>

        <div className="card">
          <h2>Reverse DCF</h2>
          {!reverseResult ? <p>Run valuation to compute implied growth.</p> : null}
          {reverseResult ? (
            <>
              <p><strong>Implied Growth Rate:</strong> {reverseResult.impliedGrowthRatePct}%</p>
              <p><strong>Model Price:</strong> {summary?.currency} {reverseResult.modelPriceAtImpliedGrowth}</p>
              <p>{growthMessage}</p>
            </>
          ) : null}

          <h3>Historical Annual Growth</h3>
          <div style={{ overflowX: "auto" }}>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 12 }}>
              <thead>
                <tr>
                  <th style={{ textAlign: "left" }}>Metric</th>
                  <th style={{ textAlign: "right" }}>1Y</th>
                  <th style={{ textAlign: "right" }}>3Y</th>
                  <th style={{ textAlign: "right" }}>5Y</th>
                  <th style={{ textAlign: "right" }}>10Y</th>
                </tr>
              </thead>
              <tbody>
                {historicalRows.map((row) => (
                  <tr key={row.metric}>
                    <td>{METRIC_LABELS[row.metric]}</td>
                    <td style={{ textAlign: "right" }}>{row.values["1Y"] ?? "-"}</td>
                    <td style={{ textAlign: "right" }}>{row.values["3Y"] ?? "-"}</td>
                    <td style={{ textAlign: "right" }}>{row.values["5Y"] ?? "-"}</td>
                    <td style={{ textAlign: "right" }}>{row.values["10Y"] ?? "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div style={{ marginTop: 10 }}>
            <button className="button" type="button" onClick={() => setShowGrowthDebug((c) => !c)}>
              {showGrowthDebug ? "Hide Growth Debug" : "Debug: Show Raw Historical Series"}
            </button>
            {showGrowthDebug && historicalSeries ? (
              <div style={{ marginTop: 10, fontSize: 12, color: "#444" }}>
                {(
                  [["eps", "EPS"], ["revenue", "Revenue"], ["operatingIncome", "Operating Income"], ["ebitda", "EBITDA"], ["fcf", "Free Cash Flow"], ["dividend", "Dividends"], ["bookValue", "Book Value"], ["price", "Price"]] as const
                ).map(([key, label]) => (
                  <div key={key} style={{ marginBottom: 8 }}>
                    <strong>{label}</strong>
                    <div>Years: {historicalSeries[key].years.join(", ") || "-"}</div>
                    <div>Values: {historicalSeries[key].values.join(", ") || "-"}</div>
                  </div>
                ))}
              </div>
            ) : null}
          </div>
          {error ? <p style={{ color: "crimson" }}>{error}</p> : null}
        </div>
      </div>
    </main>
  );
}

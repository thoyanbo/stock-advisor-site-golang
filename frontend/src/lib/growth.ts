export type GrowthMetric = "eps" | "revenue" | "operatingIncome" | "ebitda" | "fcf" | "dividend" | "bookValue" | "price";
export type BasisType = "eps" | "fcf" | "dividend";

export interface HistoricalGrowthRow {
  metric: GrowthMetric;
  values: Partial<Record<"1Y" | "3Y" | "5Y" | "10Y", number>>;
}

export interface ScenarioRecord {
  id: string;
  symbol: string;
  name: string;
  basisType: BasisType;
  discountRatePct: number;
  growthYears: number;
  growthRatePct: number;
  terminalYears: number;
  terminalRatePct: number;
  addTangibleBook: boolean;
  createdAt: string;
  updatedAt: string;
}

const HORIZON_PREFERENCE = ["10Y", "5Y", "3Y", "1Y"] as const;

function clampGrowth(v: number): number {
  if (v < 5) return 5;
  if (v > 20) return 20;
  return Math.round(v * 100) / 100;
}

function selectHistoricalGrowth(rows: HistoricalGrowthRow[], metric: string): number | null {
  for (const row of rows) {
    if (row.metric === metric) {
      for (const h of HORIZON_PREFERENCE) {
        const v = row.values[h];
        if (v !== undefined && v !== null) return v;
      }
    }
  }
  return null;
}

export function deriveGrowthRateFromHistoricalRows(
  basisType: string,
  rows: HistoricalGrowthRow[],
  fallbackPct: number
): number {
  const primaryMetric = basisType === "fcf" ? "fcf" : basisType === "dividend" ? "dividend" : "eps";
  const primary = selectHistoricalGrowth(rows, primaryMetric);
  if (primary !== null) return clampGrowth(primary);
  const eps = selectHistoricalGrowth(rows, "eps");
  if (eps !== null) return clampGrowth(eps);
  return clampGrowth(fallbackPct);
}

export function evaluateGrowthPlausibility(
  impliedGrowthPct: number,
  rows: HistoricalGrowthRow[]
): { message: string } {
  const epsCagr10 = rows.find((r) => r.metric === "eps")?.values["10Y"];
  if (epsCagr10 !== undefined && epsCagr10 !== null) {
    const diff = Math.abs(impliedGrowthPct - epsCagr10);
    if (diff < 3) {
      return { message: `Implied growth (${impliedGrowthPct}%) is close to the 10-year EPS CAGR (${epsCagr10}%).` };
    }
    if (impliedGrowthPct > epsCagr10) {
      return {
        message: `Implied growth (${impliedGrowthPct}%) exceeds the 10-year EPS CAGR (${epsCagr10}%), suggesting the market expects acceleration.`,
      };
    }
    return {
      message: `Implied growth (${impliedGrowthPct}%) is below the 10-year EPS CAGR (${epsCagr10}%), suggesting potential undervaluation.`,
    };
  }
  return { message: "Not enough historical data to compare implied growth against long-term trends." };
}

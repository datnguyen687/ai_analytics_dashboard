"use client";

import { useEffect, useMemo, useState } from "react";
import { ForecastChart } from "@/components/charts";
import { Card, DataTable, Pill, StatTile } from "@/components/ui";
import { fmtMonth } from "@/lib/analytics";
import { API_BASE, getForecast, type ForecastResponse } from "@/lib/api";
import { useMeta } from "@/lib/meta-context";

const HORIZONS = [3, 4, 6];

export default function ForecastPage() {
  const { meta } = useMeta();
  const categories = meta?.categories ?? [];
  const [category, setCategory] = useState<string | null>(null);
  const [horizon, setHorizon] = useState(4);

  const [data, setData] = useState<ForecastResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const ctrl = new AbortController();
    setLoading(true);
    setError(null);
    getForecast(category ?? "", horizon, ctrl.signal)
      .then((f) => {
        setData(f);
        setLoading(false);
      })
      .catch((e: unknown) => {
        if ((e as Error).name === "AbortError") return;
        setError((e as Error).message);
        setLoading(false);
      });
    return () => ctrl.abort();
  }, [category, horizon]);

  const label = category ?? "All categories";

  const chartData = useMemo(
    () =>
      (data?.points ?? []).map((p) => ({
        bucket: fmtMonth(p.bucket),
        actual: p.actual ?? (null as unknown as number),
        forecast: p.forecast ?? (null as unknown as number),
      })),
    [data],
  );

  if (error) {
    return (
      <div className="flex flex-col gap-4">
        <h1 className="text-lg font-semibold tracking-tight">Demand forecast</h1>
        <Card>
          <p className="text-sm font-medium text-[var(--status-critical)]">
            Couldn&apos;t reach the API ({error}).
          </p>
          <p className="mt-1 text-xs text-[var(--text-secondary)]">
            Calling <code>{API_BASE}</code>. Check that the backend is running.
          </p>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h1 className="flex items-center gap-2 text-lg font-semibold tracking-tight">
          Demand forecast
          {loading && (
            <span className="size-3.5 animate-spin rounded-full border-2 border-[var(--border)] border-t-[var(--series-1)]" />
          )}
        </h1>
        <p className="mt-0.5 text-xs text-[var(--muted)]">
          Monthly unit demand projected from 2025 history, with an inventory
          recommendation and the methodology behind it.
        </p>
      </div>

      <div className="card flex flex-col gap-3 p-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className="w-20 text-xs font-medium text-[var(--text-secondary)]">Category</span>
          <Pill active={category === null} onClick={() => setCategory(null)}>
            All
          </Pill>
          {categories.map((c) => (
            <Pill key={c} active={category === c} onClick={() => setCategory(c)}>
              {c}
            </Pill>
          ))}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="w-20 text-xs font-medium text-[var(--text-secondary)]">Horizon</span>
          {HORIZONS.map((h) => (
            <Pill key={h} active={horizon === h} onClick={() => setHorizon(h)}>
              {h} months
            </Pill>
          ))}
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatTile
          label="Avg forecast demand"
          value={data ? `${data.avgMonthlyDemand.toFixed(0)} units/mo` : "—"}
          hint={`${label} · next ${horizon} months`}
          color="var(--series-1)"
        />
        <StatTile
          label="Trend"
          value={data ? `${data.slope >= 0 ? "+" : ""}${data.slope.toFixed(1)} units/mo` : "—"}
          color={data && data.slope < 0 ? "var(--status-critical)" : "var(--status-good)"}
          hint="OLS slope on monthly units"
        />
        <StatTile
          label="Safety stock"
          value={data ? `${data.safetyStock} units` : "—"}
          hint="1.65σ, ~95% service level"
          color="var(--series-4)"
        />
        <StatTile
          label="Recommended inventory"
          value={data ? `${data.recommendedInventory} units` : "—"}
          hint="Avg demand + safety stock, per month"
          color="var(--series-2)"
        />
      </div>

      <Card
        title={`Historical vs forecast demand — ${label}`}
        subtitle="Solid line is actual units shipped; dashed line is the projection"
      >
        <ForecastChart data={chartData} />
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card title="Methodology" subtitle={data?.method ?? ""}>
          <ol className="flex flex-col gap-2">
            {(data?.explanation ?? []).map((e, i) => (
              <li key={i} className="flex gap-2 text-xs text-[var(--text-secondary)]">
                <span className="mt-0.5 grid size-4 shrink-0 place-items-center rounded-full border border-[var(--border)] text-[10px]">
                  {i + 1}
                </span>
                <span>{e}</span>
              </li>
            ))}
          </ol>
          <p className="mt-4 rounded-lg border border-[var(--border)] p-3 text-xs text-[var(--text-secondary)]">
            <strong className="text-[var(--text-primary)]">Limitation.</strong> This is
            a 400-row demo sample, so month buckets are thin and the model captures
            trend and level only — no seasonality, no promo effects. Treat it as
            directional.
          </p>
        </Card>

        <Card title="Forecast values" subtitle="The numbers behind the chart">
          <DataTable
            columns={["Month", "Actual units", "Forecast units"]}
            rows={(data?.points ?? []).map((p) => [
              fmtMonth(p.bucket),
              p.actual ?? "—",
              p.forecast ?? "—",
            ])}
            maxHeight={360}
          />
        </Card>
      </div>
    </div>
  );
}

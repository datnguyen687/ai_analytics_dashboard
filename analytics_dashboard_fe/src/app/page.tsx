"use client";

import { useEffect, useMemo, useState } from "react";
import {
  CategoryBar,
  HorizontalBar,
  RevenueTrendChart,
  Sparkline,
  StackedAreaChart,
  StatusDonut,
} from "@/components/charts";
import { FilterBar } from "@/components/filter-bar";
import { Badge, Card, DataTable, MiniBar, StatTile } from "@/components/ui";
import { DEFAULT_FILTERS, fmtMonth, fmtMoney, fmtPct } from "@/lib/analytics";
import { API_BASE, getDashboard, type DashboardResponse } from "@/lib/api";
import { CATEGORICAL, COLORS } from "@/lib/colors";

const STATUS_COLORS: Record<string, string> = {
  delivered: "var(--status-good)",
  delayed: "var(--status-critical)",
  in_transit: "var(--series-1)",
  exception: "var(--status-serious)",
  canceled: "var(--muted)",
};
const STATUS_LABEL: Record<string, string> = {
  delivered: "Delivered",
  delayed: "Delayed",
  in_transit: "In transit",
  exception: "Exception",
  canceled: "Canceled",
};

export default function DashboardPage() {
  const [filters, setFilters] = useState(DEFAULT_FILTERS);
  const [data, setData] = useState<DashboardResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // One API call per filter change — the backend assembles every aggregate.
  useEffect(() => {
    const ctrl = new AbortController();
    setLoading(true);
    setError(null);
    getDashboard(filters, ctrl.signal)
      .then((d) => {
        setData(d);
        setLoading(false);
      })
      .catch((e: unknown) => {
        if ((e as Error).name === "AbortError") return;
        setError((e as Error).message);
        setLoading(false);
      });
    return () => ctrl.abort();
  }, [filters]);

  const monthly = useMemo(
    () => (data?.revenueTrend ?? []).map((t) => ({ ...t, bucket: fmtMonth(t.bucket) })),
    [data],
  );
  const stack = useMemo(
    () => ({
      keys: data?.categoryStack.keys ?? [],
      data: (data?.categoryStack.data ?? []).map((d) => ({
        ...d,
        bucket: fmtMonth(String(d.bucket)),
      })),
    }),
    [data],
  );
  const mix = useMemo(
    () =>
      (data?.statusMix ?? []).map((s) => ({
        name: STATUS_LABEL[s.status] ?? s.status,
        value: s.count,
        color: STATUS_COLORS[s.status] ?? "var(--muted)",
      })),
    [data],
  );
  const revenueMoM = useMemo(() => {
    const rt = data?.revenueTrend ?? [];
    if (rt.length < 2) return null;
    const last = rt[rt.length - 1].revenue;
    const prev = rt[rt.length - 2].revenue;
    return prev === 0 ? null : (last - prev) / prev;
  }, [data]);

  const kpis = data?.kpis;
  const carriers = data?.carriers ?? [];
  const categories = data?.categories ?? [];
  const destinations = data?.destinations ?? [];
  const ordersSpark = monthly.map((m) => m.orders);
  const revenueSpark = monthly.map((m) => m.revenue);
  const delayedSpark = monthly.map((m) => m.delayed);
  const maxDest = Math.max(1, ...destinations.map((d) => d.orders));

  if (error) {
    return (
      <div className="flex flex-col gap-4">
        <h1 className="text-lg font-semibold tracking-tight">Operations overview</h1>
        <Card>
          <p className="text-sm font-medium text-[var(--status-critical)]">
            Couldn&apos;t reach the API ({error}).
          </p>
          <p className="mt-2 text-xs text-[var(--text-secondary)]">
            Calling <code>{API_BASE}</code>. Check that the backend is running there and
            that its <code>CORS_ORIGIN</code> allows this page&apos;s origin.
          </p>
          <p className="mt-2 text-[11px] text-[var(--muted)]">
            Override the target with <code>NEXT_PUBLIC_API_URL</code> in{" "}
            <code>.env.local</code>, then restart the dev server (Next reads it only at
            startup).
          </p>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-lg font-semibold tracking-tight">
            Operations overview
            {loading && (
              <span className="size-3.5 animate-spin rounded-full border-2 border-[var(--border)] border-t-[var(--series-1)]" />
            )}
          </h1>
          <p className="mt-0.5 text-xs text-[var(--muted)]">
            {kpis ? `${kpis.totalOrders.toLocaleString()} orders match the current filters` : "Loading…"}
          </p>
        </div>
        <FilterBar filters={filters} onChange={setFilters} />
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <StatTile
          label="Total orders"
          value={kpis ? kpis.totalOrders.toLocaleString() : "—"}
          hint={kpis ? `${fmtMoney(kpis.totalRevenue)} revenue` : ""}
          color="var(--series-1)"
          chart={<Sparkline data={ordersSpark} color={COLORS.primary} />}
        />
        <StatTile
          label="Delivered orders"
          value={kpis ? kpis.deliveredOrders.toLocaleString() : "—"}
          hint={kpis ? `${fmtPct(kpis.deliveredOrders / Math.max(1, kpis.totalOrders))} of volume` : ""}
          color="var(--status-good)"
          chart={<Sparkline data={revenueSpark} color="var(--status-good)" />}
        />
        <StatTile
          label="Delayed orders"
          value={kpis ? kpis.delayedOrders.toLocaleString() : "—"}
          hint={kpis ? `${fmtPct(kpis.delayedOrders / Math.max(1, kpis.totalOrders))} of volume` : ""}
          color="var(--status-critical)"
          chart={<Sparkline data={delayedSpark} color="var(--status-critical)" />}
        />
        <StatTile
          label="On-time delivery rate"
          value={kpis ? fmtPct(kpis.onTimeRate) : "—"}
          hint="Delivered ÷ settled orders"
          color="var(--series-3)"
          delta={
            kpis
              ? {
                  value: fmtPct(kpis.onTimeRate),
                  direction: kpis.onTimeRate >= 0.85 ? "up" : "down",
                  good: kpis.onTimeRate >= 0.85,
                }
              : undefined
          }
        />
        <StatTile
          label="Avg delivery time"
          value={kpis ? `${kpis.avgDeliveryDays.toFixed(1)} days` : "—"}
          hint="Order date → delivery date"
          color="var(--series-2)"
        />
      </div>

      <Card
        title="Revenue trend"
        subtitle="Monthly revenue with a 3-month moving average"
        action={
          revenueMoM != null && (
            <Badge tone={revenueMoM >= 0 ? "good" : "critical"}>
              {revenueMoM >= 0 ? "▲" : "▼"} {fmtPct(Math.abs(revenueMoM))} MoM
            </Badge>
          )
        }
      >
        <RevenueTrendChart data={monthly} height={320} />
      </Card>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card title="Status mix" subtitle="Share of orders by delivery status">
          <StatusDonut data={mix} total={kpis?.totalOrders ?? 0} totalLabel="orders" />
        </Card>
        <Card
          title="Order volume by category"
          subtitle="Monthly orders, stacked by top categories"
          className="lg:col-span-2"
        >
          <StackedAreaChart data={stack.data} keys={stack.keys} height={260} />
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card title="Delay rate by carrier" subtitle="Delayed ÷ settled orders — lower is better">
          <HorizontalBar
            data={carriers.map((c) => ({
              name: c.name,
              delayRate: Number((c.delayRate * 100).toFixed(1)),
            }))}
            dataKey="delayRate"
            labelKey="name"
            color={COLORS.danger}
            unit="%"
          />
        </Card>
        <Card title="Volume by product category" subtitle="Orders per category">
          <CategoryBar
            data={categories.map((c) => ({ name: c.name, orders: c.orders }))}
            palette={CATEGORICAL}
          />
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-5">
        <Card title="Top destinations" subtitle="By order count" className="lg:col-span-2">
          <ul className="flex flex-col gap-2.5 pt-1">
            {destinations.map((d, i) => (
              <li key={d.name} className="flex items-center gap-3 text-xs">
                <span className="w-4 text-[var(--muted)] tnum">{i + 1}</span>
                <span className="w-36 truncate text-[var(--text-primary)]">{d.name}</span>
                <span className="flex-1">
                  <MiniBar value={d.orders} max={maxDest} color={CATEGORICAL[i % CATEGORICAL.length]} />
                </span>
              </li>
            ))}
          </ul>
        </Card>
        <Card
          title="Carrier scorecard"
          subtitle="The rows behind the charts above"
          className="lg:col-span-3"
        >
          <DataTable
            columns={["Carrier", "Orders", "Delivered", "Delayed", "Delay rate", "Avg days", "Revenue"]}
            rows={carriers.map((c) => [
              c.name,
              c.orders,
              c.delivered,
              c.delayed,
              fmtPct(c.delayRate),
              c.avgDeliveryDays.toFixed(1),
              fmtMoney(c.revenue),
            ])}
            maxHeight={300}
          />
        </Card>
      </div>
    </div>
  );
}

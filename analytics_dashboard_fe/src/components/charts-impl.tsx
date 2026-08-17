"use client";

import { useEffect, useRef, useState } from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ComposedChart,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { CATEGORICAL, COLORS } from "@/lib/colors";
import type { ChartSpec } from "@/lib/types";

const axisProps = {
  stroke: "var(--baseline)",
  tick: { fill: "var(--muted)", fontSize: 11 },
  tickLine: false,
} as const;

const tooltipStyle = {
  contentStyle: {
    background: "var(--surface-1)",
    border: "1px solid var(--border)",
    borderRadius: 8,
    fontSize: 12,
    color: "var(--text-primary)",
    boxShadow: "0 6px 24px rgba(0,0,0,0.28)",
  },
  labelStyle: { color: "var(--text-secondary)", fontSize: 11, marginBottom: 2 },
  itemStyle: { color: "var(--text-primary)", padding: 0 },
  cursor: { fill: "var(--hover)" },
} as const;

const legendStyle = { fontSize: 11, color: "var(--text-secondary)" };

/** Measures its own width via ResizeObserver — sidesteps recharts' mount-race
 *  where a ResponsiveContainer occasionally locks to an 8×8 fallback. */
function useMeasuredWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null);
  const [width, setWidth] = useState(0);
  useEffect(() => {
    if (!ref.current) return;
    const el = ref.current;
    const ro = new ResizeObserver((entries) => {
      const w = entries[0]?.contentRect.width ?? 0;
      if (w > 0) setWidth(w);
    });
    ro.observe(el);
    setWidth(el.clientWidth);
    return () => ro.disconnect();
  }, []);
  return { ref, width };
}

export function Legendy({ items }: { items: { label: string; color: string }[] }) {
  return (
    <ul className="mb-2 flex flex-wrap gap-x-4 gap-y-1">
      {items.map((i) => (
        <li key={i.label} className="flex items-center gap-1.5 text-[11px] text-[var(--text-secondary)]">
          <span aria-hidden className="size-2 rounded-sm" style={{ background: i.color }} />
          {i.label}
        </li>
      ))}
    </ul>
  );
}

export function VolumeChart({
  data,
}: {
  data: { bucket: string; orders: number; revenue: number }[];
}) {
  return (
    <ResponsiveContainer width="100%" height={260}>
      <AreaChart data={data} margin={{ top: 4, right: 8, left: -18, bottom: 0 }}>
        <defs>
          <linearGradient id="volFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={COLORS.primary} stopOpacity={0.28} />
            <stop offset="100%" stopColor={COLORS.primary} stopOpacity={0.02} />
          </linearGradient>
        </defs>
        <CartesianGrid stroke={COLORS.grid} vertical={false} />
        <XAxis dataKey="bucket" {...axisProps} />
        <YAxis {...axisProps} width={44} />
        <Tooltip {...tooltipStyle} />
        <Area
          type="monotone"
          dataKey="orders"
          name="Orders"
          stroke={COLORS.primary}
          strokeWidth={2}
          fill="url(#volFill)"
          dot={false}
          activeDot={{ r: 4, strokeWidth: 2, stroke: COLORS.surface }}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

/**
 * Revenue trend: monthly revenue as an area, a 3-month moving-average line, and
 * a dashed average reference line — all on a single $ axis, so every mark is
 * directly comparable.
 */
export function RevenueTrendChart({
  data,
  height = 300,
}: {
  data: { bucket: string; revenue: number }[];
  height?: number;
}) {
  // 3-month trailing moving average, on the same axis as revenue.
  const withMa = data.map((d, i) => {
    const window = data.slice(Math.max(0, i - 2), i + 1);
    const ma = window.reduce((a, w) => a + w.revenue, 0) / window.length;
    return { ...d, ma3: Math.round(ma) };
  });
  const avg =
    data.length === 0 ? 0 : Math.round(data.reduce((a, d) => a + d.revenue, 0) / data.length);
  const money = (v: number) => `$${Number(v).toLocaleString()}`;

  return (
    <ResponsiveContainer width="100%" height={height}>
      <ComposedChart data={withMa} margin={{ top: 4, right: 8, left: -2, bottom: 0 }}>
        <defs>
          <linearGradient id="revFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={COLORS.primary} stopOpacity={0.3} />
            <stop offset="100%" stopColor={COLORS.primary} stopOpacity={0.02} />
          </linearGradient>
        </defs>
        <CartesianGrid stroke={COLORS.grid} vertical={false} />
        <XAxis dataKey="bucket" {...axisProps} />
        <YAxis
          {...axisProps}
          width={52}
          tickFormatter={(v: number) => `$${(v / 1000).toFixed(v < 10000 ? 1 : 0)}k`}
        />
        <Tooltip
          {...tooltipStyle}
          cursor={{ stroke: COLORS.grid }}
          formatter={(value, name) => [money(Number(value)), name]}
        />
        <Legend wrapperStyle={legendStyle} iconType="plainline" iconSize={12} />
        <ReferenceLine
          y={avg}
          stroke={COLORS.axis}
          strokeDasharray="4 4"
          label={{ value: `avg ${money(avg)}`, position: "insideTopRight", fill: COLORS.axis, fontSize: 10 }}
        />
        <Area
          type="monotone"
          dataKey="revenue"
          name="Revenue"
          stroke={COLORS.primary}
          strokeWidth={2}
          fill="url(#revFill)"
          dot={false}
          activeDot={{ r: 4, strokeWidth: 2, stroke: COLORS.surface }}
        />
        <Line
          type="monotone"
          dataKey="ma3"
          name="3-mo avg"
          stroke={COLORS.accent}
          strokeWidth={2}
          strokeDasharray="5 4"
          dot={false}
        />
      </ComposedChart>
    </ResponsiveContainer>
  );
}

export function PerformanceChart({
  data,
}: {
  data: { bucket: string; delivered: number; delayed: number }[];
}) {
  return (
    <ResponsiveContainer width="100%" height={260}>
      <BarChart data={data} margin={{ top: 4, right: 8, left: -18, bottom: 0 }} barCategoryGap="22%">
        <CartesianGrid stroke={COLORS.grid} vertical={false} />
        <XAxis dataKey="bucket" {...axisProps} />
        <YAxis {...axisProps} width={44} />
        <Tooltip {...tooltipStyle} />
        <Legend wrapperStyle={legendStyle} iconType="square" iconSize={8} />
        <Bar dataKey="delivered" name="On time" stackId="s" fill={COLORS.success} />
        <Bar
          dataKey="delayed"
          name="Delayed"
          stackId="s"
          fill={COLORS.danger}
          radius={[4, 4, 0, 0]}
        />
      </BarChart>
    </ResponsiveContainer>
  );
}

/** Stacked area — order volume split by a categorical dimension. */
export function StackedAreaChart({
  data,
  keys,
  height = 260,
}: {
  data: Record<string, string | number>[];
  keys: string[];
  height?: number;
}) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data} margin={{ top: 4, right: 8, left: -18, bottom: 0 }}>
        <defs>
          {keys.map((k, i) => (
            <linearGradient key={k} id={`grad-${i}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={CATEGORICAL[i % CATEGORICAL.length]} stopOpacity={0.55} />
              <stop offset="100%" stopColor={CATEGORICAL[i % CATEGORICAL.length]} stopOpacity={0.12} />
            </linearGradient>
          ))}
        </defs>
        <CartesianGrid stroke={COLORS.grid} vertical={false} />
        <XAxis dataKey="bucket" {...axisProps} />
        <YAxis {...axisProps} width={44} />
        <Tooltip {...tooltipStyle} cursor={{ stroke: COLORS.grid }} />
        <Legend wrapperStyle={legendStyle} iconType="square" iconSize={8} />
        {keys.map((k, i) => (
          <Area
            key={k}
            type="monotone"
            dataKey={k}
            name={k}
            stackId="a"
            stroke={CATEGORICAL[i % CATEGORICAL.length]}
            strokeWidth={2}
            fill={`url(#grad-${i})`}
          />
        ))}
      </AreaChart>
    </ResponsiveContainer>
  );
}

/** Donut for a part-to-whole status mix, with the total called out in the hole. */
export function StatusDonut({
  data,
  total,
  totalLabel,
  height = 260,
}: {
  data: { name: string; value: number; color: string }[];
  total: number;
  totalLabel: string;
  height?: number;
}) {
  const { ref, width } = useMeasuredWidth<HTMLDivElement>();
  return (
    <div className="relative" ref={ref}>
      {width > 0 && (
        <PieChart width={width} height={height}>
          <Tooltip {...tooltipStyle} cursor={{ fill: "transparent" }} />
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            innerRadius="60%"
            outerRadius="84%"
            paddingAngle={2}
            stroke={COLORS.surface}
            strokeWidth={2}
            // Explicit-width PieChart + sector animation fails to render in
            // recharts 3; the donut stays static (the trend charts still animate).
            isAnimationActive={false}
          >
            {data.map((d) => (
              <Cell key={d.name} fill={d.color} />
            ))}
          </Pie>
          <Legend wrapperStyle={legendStyle} iconType="square" iconSize={8} />
        </PieChart>
      )}
      <div
        className="pointer-events-none absolute inset-x-0 flex flex-col items-center"
        style={{ top: height * 0.36 }}
      >
        <p className="text-xl font-semibold tracking-tight tnum">{total.toLocaleString()}</p>
        <p className="text-[11px] text-[var(--muted)]">{totalLabel}</p>
      </div>
    </div>
  );
}

/** Tiny trend line for stat tiles — no axes, no chrome. */
export function Sparkline({
  data,
  color,
  height = 34,
}: {
  data: number[];
  color: string;
  height?: number;
}) {
  const points = data.map((v, i) => ({ i, v }));
  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={points} margin={{ top: 2, right: 0, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id={`spark-${color.replace(/\W/g, "")}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.35} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <Area
          type="monotone"
          dataKey="v"
          stroke={color}
          strokeWidth={2}
          fill={`url(#spark-${color.replace(/\W/g, "")})`}
          dot={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

export function HorizontalBar({
  data,
  dataKey,
  labelKey,
  color = COLORS.primary,
  height = 260,
  unit = "",
}: {
  data: Record<string, string | number>[];
  dataKey: string;
  labelKey: string;
  color?: string;
  height?: number;
  unit?: string;
}) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart
        layout="vertical"
        data={data}
        margin={{ top: 4, right: 24, left: 8, bottom: 0 }}
        barCategoryGap="22%"
      >
        <CartesianGrid stroke={COLORS.grid} horizontal={false} />
        <XAxis type="number" {...axisProps} unit={unit} />
        <YAxis type="category" dataKey={labelKey} width={96} {...axisProps} />
        <Tooltip {...tooltipStyle} />
        <Bar dataKey={dataKey} fill={color} radius={[0, 4, 4, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

export function CategoryBar({
  data,
  palette,
}: {
  data: { name: string; orders: number }[];
  palette: string[];
}) {
  return (
    <ResponsiveContainer width="100%" height={260}>
      <BarChart data={data} margin={{ top: 4, right: 8, left: -18, bottom: 0 }} barCategoryGap="22%">
        <CartesianGrid stroke={COLORS.grid} vertical={false} />
        <XAxis dataKey="name" {...axisProps} interval={0} angle={-20} textAnchor="end" height={54} />
        <YAxis {...axisProps} width={44} />
        <Tooltip {...tooltipStyle} />
        <Bar dataKey="orders" name="Orders" radius={[4, 4, 0, 0]}>
          {data.map((d, i) => (
            <Cell key={d.name} fill={palette[i % palette.length]} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
}

export function ForecastChart({
  data,
  height = 300,
}: {
  data: Record<string, string | number>[];
  height?: number;
}) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={data} margin={{ top: 4, right: 12, left: -18, bottom: 0 }}>
        <CartesianGrid stroke={COLORS.grid} vertical={false} />
        <XAxis dataKey="bucket" {...axisProps} />
        <YAxis {...axisProps} width={48} />
        <Tooltip {...tooltipStyle} cursor={{ stroke: COLORS.grid }} />
        <Legend wrapperStyle={legendStyle} iconType="plainline" iconSize={12} />
        <Line
          type="monotone"
          dataKey="actual"
          name="Actual units"
          stroke={COLORS.primary}
          strokeWidth={2}
          dot={false}
          connectNulls={false}
        />
        <Line
          type="monotone"
          dataKey="forecast"
          name="Forecast units"
          stroke={COLORS.accent}
          strokeWidth={2}
          strokeDasharray="5 4"
          dot={{ r: 3, strokeWidth: 2, stroke: COLORS.surface, fill: COLORS.accent }}
          connectNulls
        />
      </LineChart>
    </ResponsiveContainer>
  );
}

/** Renders whatever ChartSpec the AI layer returned. */
export function SpecChart({ spec }: { spec: ChartSpec }) {
  if (spec.kind === "forecast") return <ForecastChart data={spec.data} height={260} />;

  if (spec.kind === "line") {
    return (
      <ResponsiveContainer width="100%" height={240}>
        <LineChart data={spec.data} margin={{ top: 4, right: 8, left: -18, bottom: 0 }}>
          <CartesianGrid stroke={COLORS.grid} vertical={false} />
          <XAxis dataKey={spec.xKey} {...axisProps} />
          <YAxis {...axisProps} width={44} />
          <Tooltip {...tooltipStyle} cursor={{ stroke: COLORS.grid }} />
          {spec.series.map((s) => (
            <Line
              key={s.key}
              type="monotone"
              dataKey={s.key}
              name={s.label}
              stroke={s.color}
              strokeWidth={2}
              dot={false}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    );
  }

  const stacked = spec.kind === "stacked-bar";
  return (
    <ResponsiveContainer width="100%" height={240}>
      <BarChart data={spec.data} margin={{ top: 4, right: 8, left: -18, bottom: 0 }} barCategoryGap="22%">
        <CartesianGrid stroke={COLORS.grid} vertical={false} />
        <XAxis
          dataKey={spec.xKey}
          {...axisProps}
          interval={spec.data.length > 14 ? Math.floor(spec.data.length / 8) : 0}
          angle={spec.data.length > 6 ? -20 : 0}
          textAnchor={spec.data.length > 6 ? "end" : "middle"}
          height={spec.data.length > 6 ? 54 : 30}
        />
        <YAxis {...axisProps} width={44} />
        <Tooltip {...tooltipStyle} />
        {spec.series.length > 1 && (
          <Legend wrapperStyle={legendStyle} iconType="square" iconSize={8} />
        )}
        {spec.series.map((s, i) => (
          <Bar
            key={s.key}
            dataKey={s.key}
            name={s.label}
            stackId={stacked ? "s" : undefined}
            fill={s.color}
            radius={!stacked || i === spec.series.length - 1 ? [4, 4, 0, 0] : undefined}
          />
        ))}
      </BarChart>
    </ResponsiveContainer>
  );
}

"use client";

import dynamic from "next/dynamic";

/**
 * Dynamic barrel for the Recharts-based charts. Recharts is the largest
 * dependency, so it's code-split into its own chunk loaded after first paint
 * (ssr:false — the charts are client-only and measure their own width). Importers
 * keep using `@/components/charts`; the real implementations live in
 * `charts-impl.tsx`.
 */

const load = () => import("./charts-impl");

// Height-matched placeholder so lazy-loading doesn't cause layout shift.
const skeleton = (height: number) =>
  function ChartSkeleton() {
    return (
      <div
        aria-hidden
        className="animate-pulse rounded-lg bg-[var(--hover)]"
        style={{ height }}
      />
    );
  };

export const VolumeChart = dynamic(() => load().then((m) => m.VolumeChart), {
  ssr: false,
  loading: skeleton(260),
});
export const RevenueTrendChart = dynamic(() => load().then((m) => m.RevenueTrendChart), {
  ssr: false,
  loading: skeleton(320),
});
export const PerformanceChart = dynamic(() => load().then((m) => m.PerformanceChart), {
  ssr: false,
  loading: skeleton(260),
});
export const StackedAreaChart = dynamic(() => load().then((m) => m.StackedAreaChart), {
  ssr: false,
  loading: skeleton(260),
});
export const StatusDonut = dynamic(() => load().then((m) => m.StatusDonut), {
  ssr: false,
  loading: skeleton(260),
});
export const HorizontalBar = dynamic(() => load().then((m) => m.HorizontalBar), {
  ssr: false,
  loading: skeleton(260),
});
export const CategoryBar = dynamic(() => load().then((m) => m.CategoryBar), {
  ssr: false,
  loading: skeleton(260),
});
export const ForecastChart = dynamic(() => load().then((m) => m.ForecastChart), {
  ssr: false,
  loading: skeleton(300),
});
export const SpecChart = dynamic(() => load().then((m) => m.SpecChart), {
  ssr: false,
  loading: skeleton(240),
});
export const Sparkline = dynamic(() => load().then((m) => m.Sparkline), {
  ssr: false,
  loading: skeleton(34),
});
export const Legendy = dynamic(() => load().then((m) => m.Legendy), { ssr: false });

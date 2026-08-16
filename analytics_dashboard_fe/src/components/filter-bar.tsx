"use client";

import { addMonths } from "@/lib/analytics";
import { CATEGORICAL } from "@/lib/colors";
import { useMeta } from "@/lib/meta-context";
import type { Filters } from "@/lib/types";
import { DateRangePicker, MultiSelect, type RangePreset } from "./controls";

export function FilterBar({
  filters,
  onChange,
  children,
}: {
  filters: Filters;
  onChange: (f: Filters) => void;
  children?: React.ReactNode;
}) {
  const { meta } = useMeta();
  const regions = meta?.regions ?? [];
  const carriers = meta?.carriers ?? [];
  const categories = meta?.categories ?? [];
  const dateMin = meta?.dateMin ?? "";
  const dateMax = meta?.dateMax ?? "";

  const back = (months: number) => addMonths(dateMax.slice(0, 7), -(months - 1)) + "-01";
  const presets: RangePreset[] = meta
    ? [
        { label: "Full year 2025", from: dateMin, to: dateMax },
        { label: "Last 6 months", from: back(6), to: dateMax },
        { label: "Last 3 months", from: back(3), to: dateMax },
        { label: "Last month", from: back(1), to: dateMax },
      ]
    : [];

  const activeCount =
    filters.regions.length + filters.carriers.length + filters.categories.length;

  return (
    <div className="flex flex-wrap items-center gap-2">
      <DateRangePicker
        from={filters.from}
        to={filters.to}
        min={dateMin}
        max={dateMax}
        presets={presets}
        onChange={(from, to) => onChange({ ...filters, from, to })}
      />
      <MultiSelect
        label="Region"
        options={regions}
        selected={filters.regions}
        onChange={(next) => onChange({ ...filters, regions: next })}
        colors={CATEGORICAL}
      />
      <MultiSelect
        label="Carrier"
        options={carriers}
        selected={filters.carriers}
        onChange={(next) => onChange({ ...filters, carriers: next })}
      />
      <MultiSelect
        label="Category"
        options={categories}
        selected={filters.categories}
        onChange={(next) => onChange({ ...filters, categories: next })}
        colors={CATEGORICAL}
      />
      {children}
      {activeCount > 0 && (
        <button
          type="button"
          onClick={() => onChange({ ...filters, regions: [], carriers: [], categories: [] })}
          className="h-9 rounded-lg px-3 text-xs text-[var(--series-1)] transition-colors hover:bg-[var(--hover)]"
        >
          Reset {activeCount} filter{activeCount > 1 ? "s" : ""}
        </button>
      )}
    </div>
  );
}

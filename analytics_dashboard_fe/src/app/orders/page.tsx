"use client";

import { useEffect, useMemo, useState } from "react";
import { SearchInput, Select } from "@/components/controls";
import { FilterBar } from "@/components/filter-bar";
import { Card, StatusChip } from "@/components/ui";
import { DEFAULT_FILTERS, fmtMoney } from "@/lib/analytics";
import { API_BASE, getOrders, type ApiOrder, type OrderPageResponse } from "@/lib/api";

const SORTS: { value: string; label: string }[] = [
  { value: "orderDate-desc", label: "Newest first" },
  { value: "orderDate-asc", label: "Oldest first" },
  { value: "orderValue-desc", label: "Highest value" },
  { value: "transitDays-desc", label: "Slowest transit" },
  { value: "quantity-desc", label: "Largest quantity" },
];

const PAGE_SIZE = 15;

export default function OrdersPage() {
  const [filters, setFilters] = useState(DEFAULT_FILTERS);
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [status, setStatus] = useState("all");
  const [sort, setSort] = useState("orderDate-desc");
  const [page, setPage] = useState(0);

  const [data, setData] = useState<OrderPageResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Debounce the search box so we don't fire a request per keystroke.
  useEffect(() => {
    const id = setTimeout(() => setDebouncedQuery(query), 300);
    return () => clearTimeout(id);
  }, [query]);

  // Any change to a query dimension resets to the first page.
  useEffect(() => setPage(0), [filters, debouncedQuery, status, sort]);

  useEffect(() => {
    const ctrl = new AbortController();
    setLoading(true);
    setError(null);
    getOrders(
      { filters, q: debouncedQuery, status, sort, page, pageSize: PAGE_SIZE },
      ctrl.signal,
    )
      .then((res) => {
        setData(res);
        setLoading(false);
      })
      .catch((e: unknown) => {
        if ((e as Error).name === "AbortError") return;
        setError((e as Error).message);
        setLoading(false);
      });
    return () => ctrl.abort();
  }, [filters, debouncedQuery, status, sort, page]);

  const rows = data?.rows ?? [];
  const total = data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const from = total === 0 ? 0 : page * PAGE_SIZE + 1;
  const to = page * PAGE_SIZE + rows.length;

  const emptyMessage = useMemo(() => {
    if (loading) return "Loading orders…";
    if (error) return `Couldn't reach the API (${error}) — calling ${API_BASE}.`;
    return "No orders match these filters.";
  }, [loading, error]);

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h1 className="flex items-center gap-2 text-lg font-semibold tracking-tight">
          Orders
          {loading && (
            <span className="size-3.5 animate-spin rounded-full border-2 border-[var(--border)] border-t-[var(--series-1)]" />
          )}
        </h1>
        <p className="mt-0.5 text-xs text-[var(--muted)]">
          {total.toLocaleString()} orders · read-only view of the source rows
        </p>
      </div>

      <div className="flex flex-col gap-3">
        <FilterBar filters={filters} onChange={setFilters} />
        <div className="flex flex-wrap items-center gap-2">
          <SearchInput
            value={query}
            onChange={setQuery}
            placeholder="Search order ID, SKU, client, city, carrier…"
          />
          <Select
            label="Status"
            value={status}
            onChange={setStatus}
            options={[
              { value: "all", label: "All statuses" },
              { value: "delivered", label: "Delivered" },
              { value: "delayed", label: "Delayed" },
              { value: "in_transit", label: "In transit" },
              { value: "exception", label: "Exception" },
              { value: "canceled", label: "Canceled" },
            ]}
          />
          <Select label="Sort" value={sort} onChange={setSort} options={SORTS} />
        </div>
      </div>

      <Card className="p-0">
        <div className="overflow-auto">
          <table className="w-full min-w-max text-left text-xs">
            <thead className="border-b border-[var(--border)] text-[var(--text-secondary)]">
              <tr>
                {["Order", "Date", "Client", "Carrier", "Route", "SKU", "Qty", "Value", "Transit", "Status"].map(
                  (h) => (
                    <th key={h} className="px-3 py-2.5 font-medium whitespace-nowrap">
                      {h}
                    </th>
                  ),
                )}
              </tr>
            </thead>
            <tbody>
              {rows.map((o: ApiOrder) => (
                <tr
                  key={o.orderId}
                  className="border-b border-[var(--border)] transition-colors last:border-0 hover:bg-[var(--hover)]"
                >
                  <td className="px-3 py-2 font-medium tnum">{o.orderId}</td>
                  <td className="px-3 py-2 tnum text-[var(--text-secondary)]">
                    {o.orderDate.slice(0, 10)}
                  </td>
                  <td className="px-3 py-2">{o.clientId}</td>
                  <td className="px-3 py-2">{o.carrier}</td>
                  <td className="px-3 py-2 text-[var(--text-secondary)]">
                    {o.originCity} → {o.destinationCity}
                  </td>
                  <td className="px-3 py-2 tnum">{o.sku}</td>
                  <td className="px-3 py-2 tnum">{o.quantity}</td>
                  <td className="px-3 py-2 tnum">{fmtMoney(o.orderValue)}</td>
                  <td className="px-3 py-2 tnum text-[var(--text-secondary)]">
                    {o.transitDays == null ? "—" : `${o.transitDays}d`}
                  </td>
                  <td className="px-3 py-2">
                    <StatusChip status={o.status} />
                  </td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={10} className="px-3 py-10 text-center text-[var(--muted)]">
                    {emptyMessage}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="flex items-center justify-between gap-3 border-t border-[var(--border)] px-3 py-2.5 text-xs text-[var(--text-secondary)]">
          <span>
            Showing {from}–{to} of {total.toLocaleString()}
          </span>
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
              className="rounded-md border border-[var(--border)] px-2.5 py-1 transition-colors hover:bg-[var(--hover)] disabled:opacity-40"
            >
              Prev
            </button>
            <span className="px-1 tnum">
              {page + 1} / {pageCount}
            </span>
            <button
              type="button"
              onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
              disabled={page >= pageCount - 1}
              className="rounded-md border border-[var(--border)] px-2.5 py-1 transition-colors hover:bg-[var(--hover)] disabled:opacity-40"
            >
              Next
            </button>
          </div>
        </div>
      </Card>
    </div>
  );
}

"use client";

import { useEffect, useMemo, useState } from "react";
import { SearchInput, Select } from "@/components/controls";
import { FilterBar } from "@/components/filter-bar";
import { Card, StatusChip } from "@/components/ui";
import { useAuth } from "@/lib/auth-context";
import { DEFAULT_FILTERS, fmtMoney } from "@/lib/analytics";
import {
  API_BASE,
  deleteOrder,
  getOrders,
  type ApiOrder,
  type OrderPageResponse,
  type OrderWrite,
} from "@/lib/api";
import { messageForError } from "@/lib/errors";
import { EMPTY_ORDER, OrderFormModal } from "@/components/order-form";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { ImportDialog } from "@/components/import-dialog";

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
  const [reloadKey, setReloadKey] = useState(0);

  // CSV import (admin only).
  const { user } = useAuth();
  const isAdmin = user?.role === "ADMIN";
  const [importMsg, setImportMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [importOpen, setImportOpen] = useState(false);

  // Create/edit modal + delete.
  const [modal, setModal] = useState<{ initial: OrderWrite; editingId: string | null } | null>(null);
  const [rowMsg, setRowMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<ApiOrder | null>(null);
  const [deleting, setDeleting] = useState(false);

  const openCreate = () => setModal({ initial: EMPTY_ORDER, editingId: null });
  const openEdit = (o: ApiOrder) =>
    setModal({
      initial: {
        clientId: o.clientId,
        orderId: o.orderId,
        orderDate: o.orderDate.slice(0, 10),
        deliveryDate: o.deliveryDate ? o.deliveryDate.slice(0, 10) : "",
        carrier: o.carrier,
        originCity: o.originCity,
        destinationCity: o.destinationCity,
        status: o.status,
        sku: o.sku,
        category: o.category,
        quantity: o.quantity,
        unitPrice: o.unitPrice,
        orderValue: o.orderValue,
        isPromo: o.isPromo,
        promoDiscountPct: o.promoDiscountPct,
        region: o.region,
        warehouse: o.warehouse,
      },
      editingId: o.orderId,
    });

  const doDelete = async () => {
    if (!confirmDelete) return;
    setDeleting(true);
    setRowMsg(null);
    setImportMsg(null);
    try {
      await deleteOrder(confirmDelete.orderId);
      setRowMsg({ ok: true, text: `Order ${confirmDelete.orderId} deleted.` });
      setReloadKey((k) => k + 1);
      setConfirmDelete(null);
    } catch (err) {
      setRowMsg({ ok: false, text: messageForError(err) });
      setConfirmDelete(null);
    } finally {
      setDeleting(false);
    }
  };

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
  }, [filters, debouncedQuery, status, sort, page, reloadKey]);

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

          {isAdmin && (
            <div className="ml-auto flex items-center gap-2">
              <button
                type="button"
                onClick={openCreate}
                className="flex h-9 items-center gap-1.5 rounded-lg px-3 text-xs font-medium text-white"
                style={{ background: "var(--series-1)" }}
              >
                <svg viewBox="0 0 16 16" aria-hidden className="size-3.5">
                  <path d="M8 3v10M3 8h10" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
                </svg>
                Add order
              </button>
              <button
                type="button"
                onClick={() => setImportOpen(true)}
                className="flex h-9 items-center gap-2 rounded-lg border border-[var(--border)] px-3 text-xs font-medium transition-colors hover:bg-[var(--hover)]"
              >
                <svg viewBox="0 0 16 16" aria-hidden className="size-3.5">
                  <path
                    d="M8 10V2m0 0L5 5m3-3l3 3M2.5 10v2a1.5 1.5 0 001.5 1.5h8a1.5 1.5 0 001.5-1.5v-2"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
                Import CSV
              </button>
            </div>
          )}
        </div>

        {(importMsg || rowMsg) && (
          <p
            className="rounded-lg px-3 py-2 text-xs font-medium"
            style={{
              color: (importMsg ?? rowMsg)!.ok ? "var(--status-good)" : "var(--status-critical)",
              background: (importMsg ?? rowMsg)!.ok
                ? "color-mix(in srgb, var(--status-good) 10%, transparent)"
                : "color-mix(in srgb, var(--status-critical) 10%, transparent)",
            }}
          >
            {(importMsg ?? rowMsg)!.text}
          </p>
        )}
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
                {isAdmin && <th className="px-3 py-2.5 text-right font-medium">Actions</th>}
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
                  {isAdmin && (
                    <td className="px-3 py-2">
                      <div className="flex justify-end gap-1">
                        <button
                          type="button"
                          onClick={() => openEdit(o)}
                          className="rounded-md border border-[var(--border)] px-2 py-1 text-[11px] transition-colors hover:bg-[var(--hover)]"
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          onClick={() => setConfirmDelete(o)}
                          className="rounded-md border border-[var(--border)] px-2 py-1 text-[11px] text-[var(--status-critical)] transition-colors hover:bg-[color-mix(in_srgb,var(--status-critical)_10%,transparent)]"
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  )}
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={isAdmin ? 11 : 10} className="px-3 py-10 text-center text-[var(--muted)]">
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

      {modal && (
        <OrderFormModal
          initial={modal.initial}
          editingId={modal.editingId}
          onClose={() => setModal(null)}
          onSaved={(text) => {
            setRowMsg({ ok: true, text });
            setImportMsg(null);
            setReloadKey((k) => k + 1);
          }}
        />
      )}

      {importOpen && (
        <ImportDialog
          onClose={() => setImportOpen(false)}
          onDone={(msg) => {
            setImportMsg(msg);
            setRowMsg(null);
            setPage(0);
            setReloadKey((k) => k + 1);
          }}
        />
      )}

      {confirmDelete && (
        <ConfirmDialog
          title="Delete order"
          message={`Delete order ${confirmDelete.orderId}? This cannot be undone.`}
          confirmLabel="Delete"
          danger
          busy={deleting}
          onConfirm={doDelete}
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  );
}

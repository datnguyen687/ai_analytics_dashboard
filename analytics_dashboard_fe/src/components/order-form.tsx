"use client";

import { useEffect, useState } from "react";
import { createOrder, updateOrder, type OrderWrite } from "@/lib/api";
import { messageForError } from "@/lib/errors";

const STATUSES = ["delivered", "delayed", "in_transit", "exception", "canceled"];

export const EMPTY_ORDER: OrderWrite = {
  clientId: "",
  orderId: "",
  orderDate: "",
  deliveryDate: "",
  carrier: "",
  originCity: "",
  destinationCity: "",
  status: "delivered",
  sku: "",
  category: "",
  quantity: 1,
  unitPrice: 0,
  orderValue: 0,
  isPromo: false,
  promoDiscountPct: 0,
  region: "",
  warehouse: "",
};

function Field({
  label,
  children,
  wide,
}: {
  label: string;
  children: React.ReactNode;
  wide?: boolean;
}) {
  return (
    <label className={`flex flex-col gap-1 ${wide ? "sm:col-span-2" : ""}`}>
      <span className="text-[11px] font-medium text-[var(--text-secondary)]">{label}</span>
      {children}
    </label>
  );
}

const inputCls =
  "h-9 rounded-lg border border-[var(--border)] bg-transparent px-2.5 text-sm outline-none focus:border-[var(--series-1)]";

/** Create/edit modal for a single order. `editingId` set → edit mode (orderId locked). */
export function OrderFormModal({
  initial,
  editingId,
  onClose,
  onSaved,
}: {
  initial: OrderWrite;
  editingId: string | null;
  onClose: () => void;
  onSaved: (message: string) => void;
}) {
  const [form, setForm] = useState<OrderWrite>(initial);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const esc = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", esc);
    return () => document.removeEventListener("keydown", esc);
  }, [onClose]);

  const set = <K extends keyof OrderWrite>(key: K, value: OrderWrite[K]) =>
    setForm((f) => ({ ...f, [key]: value }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (saving) return;
    setSaving(true);
    setError(null);
    try {
      if (editingId) {
        await updateOrder(editingId, form);
        onSaved(`Order ${editingId} updated.`);
      } else {
        await createOrder(form);
        onSaved(`Order ${form.orderId} created.`);
      }
      onClose();
    } catch (err) {
      setError(messageForError(err));
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button type="button" aria-label="Close" onClick={onClose} className="absolute inset-0 bg-black/50" />
      <form
        onSubmit={submit}
        className="card relative z-10 max-h-[90dvh] w-full max-w-2xl overflow-auto p-5"
      >
        <h2 className="text-sm font-semibold tracking-tight">
          {editingId ? `Edit order ${editingId}` : "Add order"}
        </h2>

        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          <Field label="Order ID">
            <input
              className={inputCls}
              value={form.orderId}
              onChange={(e) => set("orderId", e.target.value)}
              disabled={!!editingId}
              required
            />
          </Field>
          <Field label="Client ID">
            <input className={inputCls} value={form.clientId} onChange={(e) => set("clientId", e.target.value)} />
          </Field>
          <Field label="Order date">
            <input
              type="date"
              className={inputCls}
              value={form.orderDate}
              onChange={(e) => set("orderDate", e.target.value)}
              required
            />
          </Field>
          <Field label="Delivery date">
            <input
              type="date"
              className={inputCls}
              value={form.deliveryDate}
              onChange={(e) => set("deliveryDate", e.target.value)}
            />
          </Field>
          <Field label="Status">
            <select className={inputCls} value={form.status} onChange={(e) => set("status", e.target.value)}>
              {STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Carrier">
            <input className={inputCls} value={form.carrier} onChange={(e) => set("carrier", e.target.value)} />
          </Field>
          <Field label="Origin city">
            <input className={inputCls} value={form.originCity} onChange={(e) => set("originCity", e.target.value)} />
          </Field>
          <Field label="Destination city">
            <input
              className={inputCls}
              value={form.destinationCity}
              onChange={(e) => set("destinationCity", e.target.value)}
            />
          </Field>
          <Field label="SKU">
            <input className={inputCls} value={form.sku} onChange={(e) => set("sku", e.target.value)} />
          </Field>
          <Field label="Category">
            <input className={inputCls} value={form.category} onChange={(e) => set("category", e.target.value)} />
          </Field>
          <Field label="Region">
            <input className={inputCls} value={form.region} onChange={(e) => set("region", e.target.value)} />
          </Field>
          <Field label="Warehouse">
            <input className={inputCls} value={form.warehouse} onChange={(e) => set("warehouse", e.target.value)} />
          </Field>
          <Field label="Quantity">
            <input
              type="number"
              min={0}
              className={inputCls}
              value={form.quantity}
              onChange={(e) => set("quantity", Number(e.target.value))}
            />
          </Field>
          <Field label="Unit price (USD)">
            <input
              type="number"
              min={0}
              step="0.01"
              className={inputCls}
              value={form.unitPrice}
              onChange={(e) => set("unitPrice", Number(e.target.value))}
            />
          </Field>
          <Field label="Order value (USD)">
            <input
              type="number"
              min={0}
              step="0.01"
              className={inputCls}
              value={form.orderValue}
              onChange={(e) => set("orderValue", Number(e.target.value))}
            />
          </Field>
          <Field label="Promo discount %">
            <input
              type="number"
              min={0}
              step="0.01"
              className={inputCls}
              value={form.promoDiscountPct}
              onChange={(e) => set("promoDiscountPct", Number(e.target.value))}
            />
          </Field>
          <label className="flex items-center gap-2 pt-5 text-xs text-[var(--text-secondary)]">
            <input
              type="checkbox"
              className="size-3.5 accent-[var(--series-1)]"
              checked={form.isPromo}
              onChange={(e) => set("isPromo", e.target.checked)}
            />
            Promotional order
          </label>
        </div>

        {error && (
          <p
            className="mt-3 rounded-lg px-3 py-2 text-xs font-medium"
            style={{
              color: "var(--status-critical)",
              background: "color-mix(in srgb, var(--status-critical) 10%, transparent)",
            }}
          >
            {error}
          </p>
        )}

        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="h-9 rounded-lg border border-[var(--border)] px-4 text-sm transition-colors hover:bg-[var(--hover)]"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={saving}
            className="h-9 rounded-lg px-4 text-sm font-medium text-white disabled:opacity-50"
            style={{ background: "var(--series-1)" }}
          >
            {saving ? "Saving…" : editingId ? "Save changes" : "Create order"}
          </button>
        </div>
      </form>
    </div>
  );
}

"use client";

import { useEffect, useRef, useState } from "react";
import { importOrders } from "@/lib/api";
import { messageForError } from "@/lib/errors";

const MAX_IMPORT_MB = 10;
const MAX_IMPORT_BYTES = MAX_IMPORT_MB * 1024 * 1024;
// Must match the backend's required columns.
const REQUIRED_COLUMNS = [
  "client_id", "order_id", "order_date", "delivery_date", "carrier", "origin_city",
  "destination_city", "status", "sku", "product_category", "quantity",
  "unit_price_usd", "order_value_usd", "is_promo", "promo_discount_pct", "region", "warehouse",
];

type Mode = "update" | "ignore" | "replaceAll";

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(2)} MB`;
}

const OPTIONS: { value: Mode; label: string; hint: string; danger?: boolean }[] = [
  { value: "update", label: "Replace duplicates", hint: "Update existing orders that share an ID; add the rest." },
  { value: "ignore", label: "Ignore duplicates", hint: "Keep existing orders; only add ones that don't exist yet." },
  { value: "replaceAll", label: "Replace all", hint: "Wipe every existing order first, then import.", danger: true },
];

/** Import panel. Opens first (before any file dialog); the admin chooses a file
 *  inside it, picks how to handle the data, then confirms. */
export function ImportDialog({
  onClose,
  onDone,
}: {
  onClose: () => void;
  onDone: (msg: { ok: boolean; text: string }) => void;
}) {
  const fileRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [rows, setRows] = useState(0);
  const [headerError, setHeaderError] = useState<string | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [mode, setMode] = useState<Mode | null>(null); // no default — must be chosen
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const esc = (e: KeyboardEvent) => e.key === "Escape" && !importing && onClose();
    document.addEventListener("keydown", esc);
    return () => document.removeEventListener("keydown", esc);
  }, [onClose, importing]);

  const chooseFile = async (picked: File | undefined) => {
    setFileError(null);
    setError(null);
    setFile(null);
    if (!picked) return;

    if (!picked.name.toLowerCase().endsWith(".csv")) {
      setFileError("Please choose a .csv file.");
      return;
    }
    if (picked.size > MAX_IMPORT_BYTES) {
      setFileError(`File is too large (${(picked.size / 1024 / 1024).toFixed(1)} MB). Max ${MAX_IMPORT_MB} MB.`);
      return;
    }

    const text = await picked.text();
    const lines = text.split(/\r?\n/).filter((l) => l.trim() !== "");
    const header = (lines[0] ?? "").split(",").map((h) => h.trim().toLowerCase());
    const missing = REQUIRED_COLUMNS.filter((c) => !header.includes(c));
    setRows(Math.max(0, lines.length - 1));
    setHeaderError(
      lines.length < 2
        ? "The file has no data rows."
        : missing.length
          ? `Missing required column${missing.length > 1 ? "s" : ""}: ${missing.join(", ")}.`
          : null,
    );
    setFile(picked);
  };

  const canImport = !!file && !headerError && mode !== null && !importing;

  const run = async () => {
    if (!file || mode === null) return;
    setImporting(true);
    setError(null);
    try {
      const res = await importOrders(file, {
        replace: mode === "replaceAll",
        onConflict: mode === "ignore" ? "ignore" : "update",
      });
      const parts = [`Imported ${res.imported.toLocaleString()} orders`];
      if (res.skipped) parts.push(`${res.skipped} duplicate${res.skipped > 1 ? "s" : ""} skipped`);
      if (res.failed) parts.push(`${res.failed} row${res.failed > 1 ? "s" : ""} skipped (errors)`);
      if (res.replaced) parts.push("existing data replaced");
      onDone({ ok: true, text: parts.join(" · ") + "." });
      onClose();
    } catch (err) {
      setError(messageForError(err));
      setImporting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <button
        type="button"
        aria-label="Close"
        onClick={() => !importing && onClose()}
        className="absolute inset-0 bg-black/50 backdrop-blur-[1px]"
      />
      <div role="dialog" aria-modal="true" className="card relative z-10 w-full max-w-md p-5">
        <h2 className="text-sm font-semibold tracking-tight">Import orders</h2>
        <p className="mt-1 text-xs text-[var(--muted)]">
          Choose a .csv file (max {MAX_IMPORT_MB} MB), pick how to handle the data, then import.
        </p>

        {/* File chooser / info */}
        <input
          ref={fileRef}
          type="file"
          accept=".csv,text/csv"
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            e.target.value = "";
            chooseFile(f);
          }}
        />

        {!file ? (
          <button
            type="button"
            onClick={() => fileRef.current?.click()}
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault();
              chooseFile(e.dataTransfer.files?.[0]);
            }}
            className="mt-4 flex w-full flex-col items-center gap-1.5 rounded-lg border border-dashed border-[var(--border)] px-3 py-6 text-center transition-colors hover:bg-[var(--hover)]"
          >
            <svg viewBox="0 0 20 20" aria-hidden className="size-5 text-[var(--muted)]">
              <path d="M10 13V4m0 0L6.5 7.5M10 4l3.5 3.5M3.5 13v2A1.5 1.5 0 0 0 5 16.5h10a1.5 1.5 0 0 0 1.5-1.5v-2" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            <span className="text-xs font-medium">Choose a CSV file</span>
            <span className="text-[11px] text-[var(--muted)]">or drag & drop it here</span>
          </button>
        ) : (
          <div className="mt-4 flex items-center gap-3 rounded-lg border border-[var(--border)] p-3">
            <span
              aria-hidden
              className="grid size-9 shrink-0 place-items-center rounded-lg"
              style={{ background: "color-mix(in srgb, var(--series-1) 14%, transparent)", color: "var(--series-1)" }}
            >
              <svg viewBox="0 0 20 20" className="size-4">
                <path d="M5 2.5h6l4 4V17a.5.5 0 0 1-.5.5h-9A.5.5 0 0 1 5 17V2.5Zm6 0V6h4" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinejoin="round" />
              </svg>
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-xs font-medium text-[var(--text-primary)]">{file.name}</p>
              <p className="text-[11px] text-[var(--muted)]">
                {formatBytes(file.size)} · {rows.toLocaleString()} data row{rows === 1 ? "" : "s"}
              </p>
            </div>
            <button
              type="button"
              onClick={() => fileRef.current?.click()}
              className="rounded-md border border-[var(--border)] px-2 py-1 text-[11px] transition-colors hover:bg-[var(--hover)]"
            >
              Change
            </button>
          </div>
        )}

        {fileError && (
          <p className="mt-2 text-xs font-medium text-[var(--status-critical)]">{fileError}</p>
        )}
        {file && headerError && (
          <p
            className="mt-2 rounded-lg px-3 py-2 text-xs font-medium"
            style={{ color: "var(--status-critical)", background: "color-mix(in srgb, var(--status-critical) 10%, transparent)" }}
          >
            {headerError}
          </p>
        )}

        {/* Options — no default; one must be chosen */}
        {file && !headerError && (
          <fieldset className="mt-4">
            <legend className="mb-1.5 text-[11px] font-medium text-[var(--text-secondary)]">
              How should the data be applied?
            </legend>
            <div className="flex flex-col gap-1.5">
              {OPTIONS.map((o) => (
                <label
                  key={o.value}
                  className={`flex cursor-pointer items-start gap-2 rounded-lg border p-2.5 transition-colors ${
                    mode === o.value ? "border-[var(--series-1)] bg-[var(--hover)]" : "border-[var(--border)] hover:bg-[var(--hover)]"
                  }`}
                >
                  <input
                    type="radio"
                    name="import-mode"
                    className="mt-0.5 accent-[var(--series-1)]"
                    checked={mode === o.value}
                    onChange={() => setMode(o.value)}
                  />
                  <span>
                    <span
                      className="block text-xs font-medium"
                      style={o.danger ? { color: "var(--status-critical)" } : undefined}
                    >
                      {o.label}
                    </span>
                    <span className="block text-[11px] text-[var(--muted)]">{o.hint}</span>
                  </span>
                </label>
              ))}
            </div>
          </fieldset>
        )}

        {error && (
          <p
            className="mt-3 rounded-lg px-3 py-2 text-xs font-medium"
            style={{ color: "var(--status-critical)", background: "color-mix(in srgb, var(--status-critical) 10%, transparent)" }}
          >
            {error}
          </p>
        )}

        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            disabled={importing}
            className="h-9 rounded-lg border border-[var(--border)] px-4 text-sm transition-colors hover:bg-[var(--hover)] disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={run}
            disabled={!canImport}
            className="flex h-9 items-center gap-2 rounded-lg px-4 text-sm font-medium text-white disabled:opacity-50"
            style={{ background: mode === "replaceAll" ? "var(--status-critical)" : "var(--series-1)" }}
          >
            {importing && <span className="size-3.5 animate-spin rounded-full border-2 border-white/40 border-t-white" />}
            {importing ? "Importing…" : "Import"}
          </button>
        </div>
      </div>
    </div>
  );
}

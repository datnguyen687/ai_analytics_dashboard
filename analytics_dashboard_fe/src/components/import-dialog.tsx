"use client";

import { useEffect, useState } from "react";
import { importOrders } from "@/lib/api";
import { messageForError } from "@/lib/errors";

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(2)} MB`;
}

/** Confirmation panel for a CSV import: shows file info and lets the admin choose
 *  how duplicates are handled before the upload actually runs. */
export function ImportDialog({
  file,
  rows,
  headerError,
  onClose,
  onDone,
}: {
  file: File;
  rows: number;
  headerError: string | null;
  onClose: () => void;
  onDone: (msg: { ok: boolean; text: string }) => void;
}) {
  const [replaceAll, setReplaceAll] = useState(false);
  const [onConflict, setOnConflict] = useState<"update" | "ignore">("update");
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const esc = (e: KeyboardEvent) => e.key === "Escape" && !importing && onClose();
    document.addEventListener("keydown", esc);
    return () => document.removeEventListener("keydown", esc);
  }, [onClose, importing]);

  const run = async () => {
    setImporting(true);
    setError(null);
    try {
      const res = await importOrders(file, { replace: replaceAll, onConflict });
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

        {/* File info */}
        <div className="mt-4 flex items-center gap-3 rounded-lg border border-[var(--border)] p-3">
          <span
            aria-hidden
            className="grid size-9 shrink-0 place-items-center rounded-lg"
            style={{ background: "color-mix(in srgb, var(--series-1) 14%, transparent)", color: "var(--series-1)" }}
          >
            <svg viewBox="0 0 20 20" className="size-4">
              <path
                d="M5 2.5h6l4 4V17a.5.5 0 0 1-.5.5h-9A.5.5 0 0 1 5 17V2.5Zm6 0V6h4"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.4"
                strokeLinejoin="round"
              />
            </svg>
          </span>
          <div className="min-w-0 flex-1">
            <p className="truncate text-xs font-medium text-[var(--text-primary)]">{file.name}</p>
            <p className="text-[11px] text-[var(--muted)]">
              {formatBytes(file.size)} · {rows.toLocaleString()} data row{rows === 1 ? "" : "s"}
            </p>
          </div>
        </div>

        {headerError ? (
          <p
            className="mt-3 rounded-lg px-3 py-2 text-xs font-medium"
            style={{ color: "var(--status-critical)", background: "color-mix(in srgb, var(--status-critical) 10%, transparent)" }}
          >
            {headerError}
          </p>
        ) : (
          <>
            {/* Duplicate handling */}
            <div className={`mt-4 ${replaceAll ? "opacity-40" : ""}`}>
              <p className="mb-1.5 text-[11px] font-medium text-[var(--text-secondary)]">
                When an order ID already exists
              </p>
              <div className="flex flex-col gap-1.5">
                <label className="flex items-center gap-2 text-xs">
                  <input
                    type="radio"
                    name="conflict"
                    className="accent-[var(--series-1)]"
                    checked={onConflict === "update"}
                    disabled={replaceAll}
                    onChange={() => setOnConflict("update")}
                  />
                  Replace it with the imported row
                </label>
                <label className="flex items-center gap-2 text-xs">
                  <input
                    type="radio"
                    name="conflict"
                    className="accent-[var(--series-1)]"
                    checked={onConflict === "ignore"}
                    disabled={replaceAll}
                    onChange={() => setOnConflict("ignore")}
                  />
                  Ignore it (keep the existing order)
                </label>
              </div>
            </div>

            <label className="mt-4 flex items-start gap-2 rounded-lg border border-[var(--border)] p-2.5 text-xs">
              <input
                type="checkbox"
                className="mt-0.5 size-3.5 accent-[var(--status-critical)]"
                checked={replaceAll}
                onChange={(e) => setReplaceAll(e.target.checked)}
              />
              <span>
                <span className="font-medium text-[var(--status-critical)]">Replace all existing orders</span>
                <span className="block text-[11px] text-[var(--muted)]">
                  Wipes the whole table first, then imports. Use to re-initialise from scratch.
                </span>
              </span>
            </label>
          </>
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
            disabled={importing || !!headerError}
            className="flex h-9 items-center gap-2 rounded-lg px-4 text-sm font-medium text-white disabled:opacity-50"
            style={{ background: replaceAll ? "var(--status-critical)" : "var(--series-1)" }}
          >
            {importing && (
              <span className="size-3.5 animate-spin rounded-full border-2 border-white/40 border-t-white" />
            )}
            {importing ? "Importing…" : replaceAll ? "Replace & import" : "Import"}
          </button>
        </div>
      </div>
    </div>
  );
}

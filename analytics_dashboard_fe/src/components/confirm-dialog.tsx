"use client";

import { useEffect } from "react";

/** Styled confirmation modal — replaces the native window.confirm(). */
export function ConfirmDialog({
  title,
  message,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  danger = false,
  busy = false,
  onConfirm,
  onCancel,
}: {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  useEffect(() => {
    const esc = (e: KeyboardEvent) => e.key === "Escape" && onCancel();
    document.addEventListener("keydown", esc);
    return () => document.removeEventListener("keydown", esc);
  }, [onCancel]);

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <button
        type="button"
        aria-label="Close"
        onClick={onCancel}
        className="absolute inset-0 bg-black/50 backdrop-blur-[1px]"
      />
      <div
        role="alertdialog"
        aria-modal="true"
        className="card relative z-10 w-full max-w-sm p-5"
      >
        <div className="flex items-start gap-3">
          <span
            aria-hidden
            className="grid size-8 shrink-0 place-items-center rounded-full"
            style={{
              color: danger ? "var(--status-critical)" : "var(--series-1)",
              background: danger
                ? "color-mix(in srgb, var(--status-critical) 12%, transparent)"
                : "color-mix(in srgb, var(--series-1) 12%, transparent)",
            }}
          >
            <svg viewBox="0 0 20 20" className="size-4">
              <path
                d="M10 6.5v4M10 13.5h.01M8.6 3.2 2.3 14a1.6 1.6 0 0 0 1.4 2.4h12.6a1.6 1.6 0 0 0 1.4-2.4L11.4 3.2a1.6 1.6 0 0 0-2.8 0Z"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </span>
          <div className="min-w-0">
            <h2 className="text-sm font-semibold tracking-tight">{title}</h2>
            <p className="mt-1 text-xs leading-relaxed text-[var(--text-secondary)]">{message}</p>
          </div>
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={onCancel}
            disabled={busy}
            className="h-9 rounded-lg border border-[var(--border)] px-4 text-sm transition-colors hover:bg-[var(--hover)] disabled:opacity-50"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={busy}
            className="h-9 rounded-lg px-4 text-sm font-medium text-white transition-opacity disabled:opacity-50"
            style={{ background: danger ? "var(--status-critical)" : "var(--series-1)" }}
          >
            {busy ? "Working…" : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}

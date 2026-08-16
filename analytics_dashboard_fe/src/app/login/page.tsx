"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { messageForError } from "@/lib/errors";

export default function LoginPage() {
  const { login } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (pending) return;
    setPending(true);
    setError(null);
    try {
      await login(username, password);
      // AppFrame redirects to "/" once the token is set.
    } catch (err) {
      setError(messageForError(err));
      setPending(false);
    }
  };

  return (
    <div className="grid min-h-dvh place-items-center bg-[var(--page)] px-4">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex items-center gap-2.5">
          <span
            aria-hidden
            className="grid size-9 place-items-center rounded-lg text-sm font-bold text-white"
            style={{ background: "linear-gradient(135deg, var(--series-1), var(--series-7))" }}
          >
            L
          </span>
          <div className="leading-tight">
            <p className="text-sm font-semibold tracking-tight">Logistics IQ</p>
            <p className="text-[11px] text-[var(--muted)]">Analytics workspace</p>
          </div>
        </div>

        <div className="card p-5">
          <h1 className="text-base font-semibold tracking-tight">Sign in</h1>
          <p className="mt-1 text-xs text-[var(--muted)]">
            Enter your credentials to access the dashboard.
          </p>

          <form onSubmit={submit} className="mt-5 flex flex-col gap-3">
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-[var(--text-secondary)]">Username</span>
              <input
                type="text"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                className="h-10 rounded-lg border border-[var(--border)] bg-transparent px-3 text-sm outline-none focus:border-[var(--series-1)]"
                placeholder="your username"
              />
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-[var(--text-secondary)]">Password</span>
              <input
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                className="h-10 rounded-lg border border-[var(--border)] bg-transparent px-3 text-sm outline-none focus:border-[var(--series-1)]"
                placeholder="••••••••"
              />
            </label>

            {error && (
              <p
                className="rounded-lg px-3 py-2 text-xs font-medium"
                style={{
                  color: "var(--status-critical)",
                  background: "color-mix(in srgb, var(--status-critical) 10%, transparent)",
                }}
              >
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={pending || !username || !password}
              className="mt-1 h-10 rounded-lg text-sm font-medium text-white transition-opacity disabled:opacity-40"
              style={{ background: "var(--series-1)" }}
            >
              {pending ? "Signing in…" : "Sign in"}
            </button>
          </form>
        </div>

        <p className="mt-4 text-center text-[11px] text-[var(--muted)]">
          Accounts are provisioned by your administrator — there is no self-service sign-up.
        </p>
      </div>
    </div>
  );
}

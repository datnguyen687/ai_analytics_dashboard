"use client";

import { useEffect, useState } from "react";
import { Badge, Card, DataTable } from "@/components/ui";
import { getAdminUsers, type AdminUser } from "@/lib/api";
import { messageForError } from "@/lib/errors";

export default function AdminPage() {
  const [users, setUsers] = useState<AdminUser[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const ctrl = new AbortController();
    getAdminUsers(ctrl.signal)
      .then((u) => {
        setUsers(u);
        setLoading(false);
      })
      .catch((e: unknown) => {
        if ((e as Error).name === "AbortError") return;
        setError(messageForError(e));
        setLoading(false);
      });
    return () => ctrl.abort();
  }, []);

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h1 className="flex items-center gap-2 text-lg font-semibold tracking-tight">
          Administration
          <Badge tone="warning">ADMIN only</Badge>
        </h1>
        <p className="mt-0.5 text-xs text-[var(--muted)]">
          Accounts and access. Visible only to accounts with the{" "}
          <code>admin:manage</code> claim.
        </p>
      </div>

      <Card title="Accounts" subtitle="Provisioned via the CLI — no self-service sign-up">
        {error ? (
          <p className="text-sm font-medium text-[var(--status-critical)]">{error}</p>
        ) : loading ? (
          <p className="text-sm text-[var(--muted)]">Loading accounts…</p>
        ) : (
          <DataTable
            columns={["Username", "Role", "Created"]}
            rows={(users ?? []).map((u) => [
              u.username,
              u.role,
              new Date(u.createdAt).toLocaleDateString(),
            ])}
            maxHeight={420}
          />
        )}
      </Card>

      <Card title="Create an account">
        <p className="text-xs text-[var(--text-secondary)]">
          Accounts are seeded from the backend. Passwords are hashed with bcrypt
          (cost 12, per-password salt) before storage — plaintext is never persisted.
        </p>
        <pre className="mt-3 overflow-auto rounded-lg border border-[var(--border)] bg-[var(--page)] p-3 text-[11px] leading-relaxed">
{`cd analytics_dashboard_be
DATABASE_URL="postgres://…/analytics_dashboard?sslmode=disable" \\
  ./scripts/create-user.sh <username> <password> ADMIN`}
        </pre>
      </Card>
    </div>
  );
}

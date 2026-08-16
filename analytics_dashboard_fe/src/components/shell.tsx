"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ReactNode, useEffect, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { MetaProvider } from "@/lib/meta-context";
import { LoadingBar } from "./loading-bar";

interface NavItem {
  href: string;
  label: string;
  hint: string;
  icon: ReactNode;
  color: string;
}

const icon = (d: string) => (
  <svg viewBox="0 0 20 20" aria-hidden className="size-4 shrink-0">
    <path d={d} fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

const GROUPS: { title: string; items: NavItem[] }[] = [
  {
    title: "Analytics",
    items: [
      {
        href: "/",
        label: "Overview",
        hint: "KPIs & charts",
        color: "var(--series-1)",
        icon: icon("M3 11h4v6H3zM8 5h4v12H8zM13 8h4v9h-4z"),
      },
      {
        href: "/orders",
        label: "Orders",
        hint: "Detailed table",
        color: "var(--series-3)",
        icon: icon("M3 5h14M3 10h14M3 15h14"),
      },
    ],
  },
  {
    title: "Intelligence",
    items: [
      {
        href: "/ask",
        label: "Ask AI",
        hint: "Natural language",
        color: "var(--series-7)",
        icon: icon("M4 4h12v9H8l-4 3z"),
      },
      {
        href: "/forecast",
        label: "Forecast",
        hint: "Demand planning",
        color: "var(--series-2)",
        icon: icon("M3 14l4-5 3 3 6-8M13 4h4v4"),
      },
    ],
  },
];

// Shown only to ADMIN accounts.
const ADMIN_GROUP: { title: string; items: NavItem[] } = {
  title: "Administration",
  items: [
    {
      href: "/admin",
      label: "Admin",
      hint: "Accounts & access",
      color: "var(--status-warning)",
      icon: icon("M10 2l6 3v5c0 3.5-2.5 6.5-6 7.5C6.5 16.5 4 13.5 4 10V5l6-3z"),
    },
  ],
};

export function Shell({ children }: { children: ReactNode }) {
  const path = usePathname();
  const { user, logout } = useAuth();
  const [open, setOpen] = useState(true);
  const [mobileOpen, setMobileOpen] = useState(false);

  // ADMIN accounts additionally see the Administration group.
  const groups = user?.role === "ADMIN" ? [...GROUPS, ADMIN_GROUP] : GROUPS;

  useEffect(() => {
    const saved = localStorage.getItem("sidebar-open");
    if (saved !== null) setOpen(saved === "1");
  }, []);

  useEffect(() => {
    localStorage.setItem("sidebar-open", open ? "1" : "0");
  }, [open]);

  useEffect(() => setMobileOpen(false), [path]);

  const width = open ? "16rem" : "4.25rem";

  return (
    <MetaProvider>
    <div className="min-h-dvh">
      <LoadingBar />
      <aside
        className={`fixed inset-y-0 left-0 z-40 flex flex-col border-r border-[var(--border)] bg-[var(--surface-1)] transition-[width,transform] duration-200 ${
          mobileOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
        }`}
        style={{ width }}
      >
        <div className="flex h-14 items-center gap-2.5 px-3">
          <span
            aria-hidden
            className="grid size-8 shrink-0 place-items-center rounded-lg text-[13px] font-bold text-white"
            style={{ background: "linear-gradient(135deg, var(--series-1), var(--series-7))" }}
          >
            L
          </span>
          {open && (
            <div className="min-w-0 leading-tight">
              <p className="truncate text-sm font-semibold tracking-tight">Logistics IQ</p>
              <p className="truncate text-[10px] text-[var(--muted)]">Analytics workspace</p>
            </div>
          )}
        </div>

        <nav className="flex-1 overflow-y-auto px-2 py-2">
          {groups.map((g) => (
            <div key={g.title} className="mb-4">
              {open && (
                <p className="mb-1 px-2 text-[10px] font-semibold uppercase tracking-wider text-[var(--muted)]">
                  {g.title}
                </p>
              )}
              <ul className="flex flex-col gap-0.5">
                {g.items.map((n) => {
                  const active = path === n.href;
                  return (
                    <li key={n.href}>
                      <Link
                        href={n.href}
                        title={open ? undefined : n.label}
                        className={`group relative flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm transition-colors ${
                          active
                            ? "bg-[var(--hover)] font-medium text-[var(--text-primary)]"
                            : "text-[var(--text-secondary)] hover:bg-[var(--hover)]"
                        }`}
                      >
                        {active && (
                          <span
                            aria-hidden
                            className="absolute inset-y-1.5 left-0 w-[3px] rounded-r"
                            style={{ background: n.color }}
                          />
                        )}
                        <span style={{ color: active ? n.color : "var(--muted)" }}>{n.icon}</span>
                        {open && (
                          <span className="min-w-0 leading-tight">
                            <span className="block truncate">{n.label}</span>
                            <span className="block truncate text-[10px] text-[var(--muted)]">
                              {n.hint}
                            </span>
                          </span>
                        )}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </nav>

        {user && (
          <div className="border-t border-[var(--border)] p-2">
            <div className={`flex items-center gap-2.5 rounded-lg px-2.5 py-2 ${open ? "" : "justify-center"}`}>
              <span
                aria-hidden
                className="grid size-7 shrink-0 place-items-center rounded-full text-[11px] font-semibold text-white"
                style={{ background: "var(--series-7)" }}
                title={user.username}
              >
                {user.username.slice(0, 2).toUpperCase()}
              </span>
              {open && (
                <div className="min-w-0 flex-1 leading-tight">
                  <p className="truncate text-xs font-medium">{user.username}</p>
                  <p className="text-[10px] text-[var(--muted)]">{user.role}</p>
                </div>
              )}
              {open && (
                <button
                  type="button"
                  onClick={logout}
                  title="Sign out"
                  className="rounded-md p-1.5 text-[var(--muted)] transition-colors hover:bg-[var(--hover)] hover:text-[var(--status-critical)]"
                >
                  <svg viewBox="0 0 20 20" aria-hidden className="size-4">
                    <path d="M8 5V4a1 1 0 011-1h6a1 1 0 011 1v12a1 1 0 01-1 1H9a1 1 0 01-1-1v-1M11 10H3m0 0l3-3m-3 3l3 3" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </button>
              )}
            </div>
          </div>
        )}

        <div className="border-t border-[var(--border)] p-2">
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            className="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-xs text-[var(--text-secondary)] transition-colors hover:bg-[var(--hover)]"
          >
            <svg viewBox="0 0 20 20" aria-hidden className="size-4 shrink-0">
              <rect x="2.5" y="3.5" width="15" height="13" rx="2" fill="none" stroke="currentColor" strokeWidth="1.6" />
              <path d="M8 3.5v13" stroke="currentColor" strokeWidth="1.6" />
            </svg>
            {open && <span>Collapse sidebar</span>}
          </button>
        </div>
      </aside>

      {mobileOpen && (
        <button
          type="button"
          aria-label="Close navigation"
          onClick={() => setMobileOpen(false)}
          className="fixed inset-0 z-30 bg-black/40 lg:hidden"
        />
      )}

      <div className="transition-[padding] duration-200 lg:pl-[var(--sidebar-w)]" style={{ ["--sidebar-w" as string]: width }}>
        <header className="sticky top-0 z-20 flex h-14 items-center gap-3 border-b border-[var(--border)] bg-[var(--page)]/85 px-4 backdrop-blur sm:px-6">
          <button
            type="button"
            onClick={() => setMobileOpen(true)}
            className="rounded-lg border border-[var(--border)] p-1.5 lg:hidden"
            aria-label="Open navigation"
          >
            <svg viewBox="0 0 20 20" aria-hidden className="size-4">
              <path d="M3 5h14M3 10h14M3 15h14" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
            </svg>
          </button>
          <p className="text-sm font-medium tracking-tight">
            {groups.flatMap((g) => g.items).find((i) => i.href === path)?.label ?? "Dashboard"}
          </p>
          <span className="ml-auto hidden items-center gap-2 text-[11px] text-[var(--muted)] sm:flex">
            <span className="size-1.5 rounded-full" style={{ background: "var(--status-good)" }} />
            Demo dataset · 400 orders · 2025
          </span>
        </header>

        <main className="px-4 py-6 sm:px-6">{children}</main>

        <footer className="px-4 pb-8 text-[11px] text-[var(--muted)] sm:px-6">
          Figures are computed by the Go/Postgres API and served over REST.
        </footer>
      </div>
    </div>
    </MetaProvider>
  );
}

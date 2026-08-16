"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";
import { useAuth } from "@/lib/auth-context";
import { Shell } from "./shell";

/** Gates the app: unauthenticated users are redirected to /login; the login
 *  route renders bare (no sidebar), everything else renders inside the Shell. */
export function AppFrame({ children }: { children: ReactNode }) {
  const { ready, token, user } = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  const isLogin = pathname === "/login";
  const isAdminRoute = pathname.startsWith("/admin");

  useEffect(() => {
    if (!ready) return;
    if (!token && !isLogin) router.replace("/login");
    if (token && isLogin) router.replace("/");
    // Non-admins can't reach admin routes (the API also enforces this).
    if (token && isAdminRoute && user?.role !== "ADMIN") router.replace("/");
  }, [ready, token, user, isLogin, isAdminRoute, router]);

  // Until we've read localStorage, render nothing to avoid a flash of the wrong view.
  if (!ready) {
    return <div className="min-h-dvh bg-[var(--page)]" />;
  }
  if (isLogin) return <>{children}</>;
  if (!token) return <div className="min-h-dvh bg-[var(--page)]" />; // redirecting
  return <Shell>{children}</Shell>;
}

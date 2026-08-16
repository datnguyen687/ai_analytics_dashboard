"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  login as apiLogin,
  setAuthToken,
  setUnauthorizedHandler,
  type AuthUser,
} from "./api";

const STORAGE_KEY = "auth_token";

interface AuthState {
  token: string | null;
  user: AuthUser | null;
  ready: boolean; // localStorage has been read (avoids SSR/hydration flash)
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

/** Decodes the JWT payload (unverified — the backend verifies on every call;
 *  this is only for reading username/role/exp for the UI). */
function decode(token: string): { username?: string; role?: AuthUser["role"]; exp?: number } | null {
  try {
    const payload = token.split(".")[1];
    return JSON.parse(atob(payload.replace(/-/g, "+").replace(/_/g, "/")));
  } catch {
    return null;
  }
}

function isExpired(token: string): boolean {
  const p = decode(token);
  return !p?.exp || p.exp * 1000 <= Date.now();
}

function userFrom(token: string): AuthUser | null {
  const p = decode(token);
  if (!p?.username || !p?.role) return null;
  return { username: p.username, role: p.role, claims: [] };
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(null);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [ready, setReady] = useState(false);

  const logout = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY);
    setAuthToken(null);
    setToken(null);
    setUser(null);
  }, []);

  // Read the stored token once on mount; drop it if already expired.
  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && !isExpired(stored)) {
      setAuthToken(stored);
      setToken(stored);
      setUser(userFrom(stored));
    } else if (stored) {
      localStorage.removeItem(STORAGE_KEY);
    }
    setReady(true);
  }, []);

  // Any 401 on an authenticated request ends the session.
  useEffect(() => {
    setUnauthorizedHandler(logout);
    return () => setUnauthorizedHandler(null);
  }, [logout]);

  const login = useCallback(async (username: string, password: string) => {
    const res = await apiLogin(username.trim().toLowerCase(), password);
    localStorage.setItem(STORAGE_KEY, res.token);
    setAuthToken(res.token);
    setToken(res.token);
    setUser(res.user);
  }, []);

  const value = useMemo<AuthState>(
    () => ({ token, user, ready, login, logout }),
    [token, user, ready, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}

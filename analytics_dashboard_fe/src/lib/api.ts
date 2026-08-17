import { ApiError } from "./errors";
import type { AnswerResult, Filters } from "./types";

export const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// --- auth wiring (set by the AuthProvider) ---
let authToken: string | null = null;
let onUnauthorized: (() => void) | null = null;

export function setAuthToken(token: string | null) {
  authToken = token;
}

/** Registered by the AuthProvider; called when an authenticated request is
 *  rejected as unauthorized (expired/invalid token) so the app can sign out. */
export function setUnauthorizedHandler(fn: (() => void) | null) {
  onUnauthorized = fn;
}
const BASE = API_BASE;

export interface ApiKpis {
  totalOrders: number;
  deliveredOrders: number;
  delayedOrders: number;
  onTimeRate: number;
  avgDeliveryDays: number;
  totalRevenue: number;
}

export interface ApiTimePoint {
  bucket: string;
  orders: number;
  delivered: number;
  delayed: number;
  revenue: number;
}

export interface ApiStatusCount {
  status: string;
  count: number;
}

export interface ApiBreakdownRow {
  name: string;
  orders: number;
  delivered: number;
  delayed: number;
  delayRate: number;
  avgDeliveryDays: number;
  revenue: number;
}

export interface ApiCategoryStack {
  keys: string[];
  data: Record<string, string | number>[];
}

export interface DashboardResponse {
  filters: Filters;
  kpis: ApiKpis;
  revenueTrend: ApiTimePoint[];
  statusMix: ApiStatusCount[];
  categoryStack: ApiCategoryStack;
  carriers: ApiBreakdownRow[];
  categories: ApiBreakdownRow[];
  destinations: ApiBreakdownRow[];
}

export interface MetaResponse {
  carriers: string[];
  regions: string[];
  categories: string[];
  statuses: string[];
  dateMin: string;
  dateMax: string;
}

function filtersToQuery(f: Filters): string {
  const p = new URLSearchParams();
  if (f.from) p.set("from", f.from);
  if (f.to) p.set("to", f.to);
  if (f.regions.length) p.set("regions", f.regions.join(","));
  if (f.carriers.length) p.set("carriers", f.carriers.join(","));
  if (f.categories.length) p.set("categories", f.categories.join(","));
  return p.toString();
}

// --- global in-flight tracking, so the UI can show a loading indicator ---
let pending = 0;
const loadingListeners = new Set<(active: boolean) => void>();

function emitLoading() {
  const active = pending > 0;
  loadingListeners.forEach((cb) => cb(active));
}

/** Subscribe to "is any API request in flight". Returns an unsubscribe fn. */
export function onLoadingChange(cb: (active: boolean) => void): () => void {
  loadingListeners.add(cb);
  cb(pending > 0);
  return () => loadingListeners.delete(cb);
}

async function request<T>(input: RequestInfo, init: RequestInit = {}): Promise<T> {
  pending += 1;
  emitLoading();
  const hadToken = authToken != null;
  const headers = new Headers(init.headers);
  if (authToken) headers.set("Authorization", `Bearer ${authToken}`);
  try {
    const res = await fetch(input, { ...init, headers });
    if (!res.ok) {
      // Parse the backend's {code, message} envelope.
      let code = "INTERNAL_ERROR";
      let message = res.statusText;
      try {
        const body = await res.json();
        if (body?.code) code = body.code;
        if (body?.message) message = body.message;
      } catch {
        /* non-JSON error body */
      }
      // An authenticated request rejected as unauthorized → session ended.
      if (res.status === 401 && hadToken) onUnauthorized?.();
      throw new ApiError(code, res.status, message);
    }
    return (await res.json()) as T;
  } catch (err) {
    // Network failure (server down / CORS) — surface as a typed error.
    if (err instanceof ApiError) throw err;
    if ((err as Error).name === "AbortError") throw err;
    throw new ApiError("NETWORK_ERROR", 0, (err as Error).message);
  } finally {
    pending -= 1;
    emitLoading();
  }
}

async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  return request<T>(`${BASE}${path}`, { signal });
}

async function postJSON<T>(path: string, body: unknown, signal?: AbortSignal): Promise<T> {
  return request<T>(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal,
  });
}

export function getDashboard(f: Filters, signal?: AbortSignal): Promise<DashboardResponse> {
  return getJSON<DashboardResponse>(`/api/v1/dashboard?${filtersToQuery(f)}`, signal);
}

export function getMeta(signal?: AbortSignal): Promise<MetaResponse> {
  return getJSON<MetaResponse>(`/api/v1/meta`, signal);
}

export interface ApiOrder {
  clientId: string;
  orderId: string;
  orderDate: string;
  deliveryDate: string | null;
  carrier: string;
  originCity: string;
  destinationCity: string;
  status: string;
  sku: string;
  category: string;
  quantity: number;
  unitPrice: number;
  orderValue: number;
  isPromo: boolean;
  promoDiscountPct: number;
  region: string;
  warehouse: string;
  transitDays: number | null;
}

export interface OrderPageResponse {
  rows: ApiOrder[];
  total: number;
  page: number;
  pageSize: number;
}

export interface OrdersParams {
  filters: Filters;
  q?: string;
  status?: string;
  sort?: string;
  page: number;
  pageSize: number;
}

export function getOrders(p: OrdersParams, signal?: AbortSignal): Promise<OrderPageResponse> {
  const params = new URLSearchParams(filtersToQuery(p.filters));
  if (p.q) params.set("q", p.q);
  if (p.status && p.status !== "all") params.set("status", p.status);
  if (p.sort) params.set("sort", p.sort);
  params.set("page", String(p.page));
  params.set("pageSize", String(p.pageSize));
  return getJSON<OrderPageResponse>(`/api/v1/orders?${params.toString()}`, signal);
}

export function postAsk(question: string, signal?: AbortSignal): Promise<AnswerResult> {
  return postJSON<AnswerResult>(`/api/v1/ask`, { question }, signal);
}

export async function getSuggestions(signal?: AbortSignal): Promise<string[]> {
  const r = await getJSON<{ suggestions: string[] }>(`/api/v1/suggestions`, signal);
  return r.suggestions ?? [];
}

export interface ForecastPointResp {
  bucket: string;
  actual: number | null;
  forecast: number | null;
}

export interface ForecastResponse {
  category: string;
  horizonMonths: number;
  method: string;
  points: ForecastPointResp[];
  forecastUnits: number[];
  avgMonthlyDemand: number;
  slope: number;
  safetyStock: number;
  recommendedInventory: number;
  explanation: string[];
}

export function getForecast(
  category: string,
  horizon: number,
  signal?: AbortSignal,
): Promise<ForecastResponse> {
  const params = new URLSearchParams();
  if (category) params.set("category", category);
  params.set("horizon", String(horizon));
  return getJSON<ForecastResponse>(`/api/v1/forecast?${params.toString()}`, signal);
}

export type Role = "USER" | "ADMIN";

export interface AuthUser {
  username: string;
  role: Role;
  claims: string[];
}

export interface LoginResponse {
  token: string;
  user: AuthUser;
}

export function login(username: string, password: string, signal?: AbortSignal): Promise<LoginResponse> {
  return postJSON<LoginResponse>(`/api/v1/auth/login`, { username, password }, signal);
}

export function getMe(signal?: AbortSignal): Promise<AuthUser> {
  return getJSON<AuthUser>(`/api/v1/auth/me`, signal);
}

export interface AdminUser {
  id: number;
  username: string;
  role: Role;
  createdAt: string;
}

export async function getAdminUsers(signal?: AbortSignal): Promise<AdminUser[]> {
  const r = await getJSON<{ users: AdminUser[] }>(`/api/v1/admin/users`, signal);
  return r.users ?? [];
}

export interface ImportResult {
  imported: number;
  failed: number;
  errors: string[];
  replaced: boolean;
}

export function importOrders(
  file: File,
  replace: boolean,
  signal?: AbortSignal,
): Promise<ImportResult> {
  const form = new FormData();
  form.append("file", file);
  if (replace) form.append("replace", "true");
  // FormData sets its own multipart Content-Type; request() adds the auth header.
  return request<ImportResult>(`${BASE}/api/v1/admin/orders/import`, {
    method: "POST",
    body: form,
    signal,
  });
}

export interface OrderWrite {
  clientId: string;
  orderId: string;
  orderDate: string; // YYYY-MM-DD
  deliveryDate: string; // YYYY-MM-DD or ""
  carrier: string;
  originCity: string;
  destinationCity: string;
  status: string;
  sku: string;
  category: string;
  quantity: number;
  unitPrice: number;
  orderValue: number;
  isPromo: boolean;
  promoDiscountPct: number;
  region: string;
  warehouse: string;
}

export function createOrder(o: OrderWrite, signal?: AbortSignal): Promise<ApiOrder> {
  return postJSON<ApiOrder>(`/api/v1/admin/orders`, o, signal);
}

export function updateOrder(orderId: string, o: OrderWrite, signal?: AbortSignal): Promise<ApiOrder> {
  return request<ApiOrder>(`${BASE}/api/v1/admin/orders/${encodeURIComponent(orderId)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(o),
    signal,
  });
}

export function deleteOrder(orderId: string, signal?: AbortSignal): Promise<{ deleted: string }> {
  return request<{ deleted: string }>(`${BASE}/api/v1/admin/orders/${encodeURIComponent(orderId)}`, {
    method: "DELETE",
    signal,
  });
}

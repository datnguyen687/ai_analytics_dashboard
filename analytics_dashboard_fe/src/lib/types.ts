export type OrderStatus =
  | "delivered"
  | "delayed"
  | "in_transit"
  | "exception"
  | "canceled";

export interface Order {
  clientId: string;
  orderId: string;
  orderDate: string;
  deliveryDate: string | null;
  carrier: string;
  originCity: string;
  destinationCity: string;
  status: OrderStatus;
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

/** Filters the UI can express. Mirrors the query contract the backend will expose. */
export interface Filters {
  from: string;
  to: string;
  regions: string[];
  carriers: string[];
  categories: string[];
}

export interface Kpis {
  totalOrders: number;
  deliveredOrders: number;
  delayedOrders: number;
  onTimeRate: number;
  avgDeliveryDays: number;
  totalRevenue: number;
}

export type ChartKind = "line" | "bar" | "stacked-bar" | "pie" | "forecast";

/**
 * Structured interpretation produced by the AI layer. The AI only ever emits
 * this plan — the numbers themselves always come from a deterministic tool.
 */
export interface QueryPlan {
  tool: "analytics.query" | "forecast.demand";
  intent: string;
  metrics: string[];
  dimensions: string[];
  filters: Record<string, string>;
  granularity?: "day" | "week" | "month";
  sort?: string;
  limit?: number;
  notes?: string[];
}

export interface ChartSpec {
  kind: ChartKind;
  title: string;
  xKey: string;
  series: { key: string; label: string; color: string }[];
  data: Record<string, string | number>[];
}

export interface AnswerResult {
  answer: string;
  plan: QueryPlan;
  chart?: ChartSpec;
  table?: { columns: string[]; rows: (string | number)[][] };
  confidence: "high" | "medium" | "low";
}

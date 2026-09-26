const BASE = "/api";

let token: string | null = null;

export function setToken(t: string | null) {
  token = t;
  if (t) {
    localStorage.setItem("pg_token", t);
  } else {
    localStorage.removeItem("pg_token");
  }
}

export function getToken(): string | null {
  if (!token) {
    token = localStorage.getItem("pg_token");
  }
  return token;
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? "request failed");
  }
  return res.json();
}

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, body),
  put: <T>(path: string, body?: unknown) => request<T>("PUT", path, body),
  delete: <T>(path: string) => request<T>("DELETE", path),
};

// ---- Types (shared with backend) ----

export interface Store {
  id: string;
  name: string;
  webhook_url: string;
  active: boolean;
  created_at: string;
}

export interface Transaction {
  transaction_id: string;
  amount: number;
  unique_amount: number;
  reference: string;
  status: "pending" | "settlement" | "expired" | "cancelled";
  qr_string: string;
  provider: string;
  expires_at: string;
  paid_at: string | null;
  created_at: string;
}

export interface WebhookDelivery {
  id: string;
  transaction_id: string;
  store_id: string;
  event: string;
  url: string;
  status: "pending" | "success" | "failed";
  status_code: number;
  response: string;
  attempt: number;
  next_retry_at: string | null;
  created_at: string;
}

export interface ProviderStatus {
  authenticated: boolean;
}
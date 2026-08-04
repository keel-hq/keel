import type {
  Approval,
  AuditResponse,
  Resource,
  Stats,
  TrackedImage,
  UserInfo,
} from "@/types"

const TOKEN_KEY = "keel.auth.token"
export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}
export const tokenStore = {
  get: () => localStorage.getItem(TOKEN_KEY),
  set: (token: string) => localStorage.setItem(TOKEN_KEY, token),
  clear: () => localStorage.removeItem(TOKEN_KEY),
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body) headers.set("Content-Type", "application/json")
  const token = tokenStore.get()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  const response = await fetch(`/v1/${path}`, { ...init, headers })
  if (response.status === 401) {
    tokenStore.clear()
    window.dispatchEvent(new Event("keel:unauthorized"))
  }
  if (!response.ok)
    throw new ApiError(
      response.status,
      (await response.text()) || response.statusText
    )
  const text = await response.text()
  return (text ? JSON.parse(text) : undefined) as T
}

export const api = {
  async login(username: string, password: string) {
    const response = await fetch("/v1/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    })
    if (!response.ok)
      throw new ApiError(
        response.status,
        (await response.text()) || response.statusText
      )
    const body = (await response.json()) as { token?: string }
    const token =
      body.token ||
      response.headers.get("Authorization")?.replace(/^Bearer\s+/i, "")
    if (!token)
      throw new ApiError(
        response.status,
        "Authentication response did not include a token"
      )
    tokenStore.set(token)
  },
  user: () => request<UserInfo>("auth/user"),
  logout: () =>
    request<Record<string, never>>("auth/logout", { method: "POST" }),
  resources: () =>
    request<Resource[] | null>("resources").then((value) => value ?? []),
  approvals: () => request<Approval[]>("approvals"),
  tracked: () =>
    request<TrackedImage[] | null>("tracked").then((value) => value ?? []),
  stats: () => request<Stats[] | null>("stats").then((value) => value ?? []),
  audit: () => request<AuditResponse>("audit?filter=*&limit=0&offset=0"),
  setPolicy: (payload: {
    identifier: string
    provider: string
    policy: string
  }) => request("policies", { method: "PUT", body: JSON.stringify(payload) }),
  setTracking: (payload: {
    identifier: string
    provider: string
    trigger: string
  }) => request("tracked", { method: "PUT", body: JSON.stringify(payload) }),
  setApprovalCount: (payload: {
    identifier: string
    provider: string
    votesRequired: number
  }) => request("approvals", { method: "PUT", body: JSON.stringify(payload) }),
  updateApproval: (payload: {
    id: string
    identifier: string
    action: string
    voter: string
  }) => request("approvals", { method: "POST", body: JSON.stringify(payload) }),
}

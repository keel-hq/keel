export interface UserInfo {
  id: string
  name: string
  username: string
  role_id: string
  auth_mode?: "legacy" | "basic" | "external-proxy"
  logout_url?: string
}
export interface Resource {
  provider: string
  identifier: string
  name: string
  namespace: string
  kind: string
  policy: string
  images: string[]
  labels: Record<string, string>
  annotations: Record<string, string>
  status: { replicas: number; availableReplicas: number }
}
export interface Approval {
  id: string
  provider: string
  identifier: string
  message: string
  currentVersion: string
  newVersion: string
  votesRequired: number
  votesReceived: number
  rejected: boolean
  archived: boolean
  deadline: string
  createdAt: string
  updatedAt: string
}
export interface TrackedImage {
  image: string
  trigger: string
  pollSchedule: string
  provider: string
  namespace: string
  policy: string
  registry: string
}
export interface AuditLog {
  id: string
  createdAt: string
  action: string
  resourceKind: string
  identifier: string
  metadata: Record<string, string>
}
export interface AuditResponse {
  data: AuditLog[]
  total: number
  limit: number
  offset: number
}
export interface Stats {
  date: string
  approved: number
  rejected: number
  updates: number
  webhooks: number
}

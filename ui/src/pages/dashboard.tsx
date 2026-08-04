import { useCallback, useEffect, useMemo, useState } from "react"
import { ChevronDown, Minus, Pause, Plus, RefreshCw } from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { Approval, Resource, Stats } from "@/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import {
  EmptyRow,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/data-table"
import { StatCard } from "@/components/stat-card"
import { PageHeading } from "@/components/page-heading"

type PolicyDialog = { resource: Resource; kind: "glob" | "regexp" } | null
export function DashboardPage() {
  const [resources, setResources] = useState<Resource[]>([])
  const [approvals, setApprovals] = useState<Approval[]>([])
  const [stats, setStats] = useState<Stats[]>([])
  const [filter, setFilter] = useState("")
  const [loading, setLoading] = useState(true)
  const [policyDialog, setPolicyDialog] = useState<PolicyDialog>(null)
  const [policyInput, setPolicyInput] = useState("")
  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [nextResources, nextApprovals, nextStats] = await Promise.all([
        api.resources(),
        api.approvals(),
        api.stats(),
      ])
      setResources(nextResources)
      setApprovals(nextApprovals)
      setStats(nextStats)
    } catch (error) {
      toast.error("Could not load dashboard", { description: String(error) })
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    void load()
  }, [load])
  const pending = approvals.filter(
    (item) =>
      !item.rejected &&
      !item.archived &&
      item.votesReceived < item.votesRequired
  )
  const approved = approvals.filter(
    (item) => !item.rejected && item.votesReceived >= item.votesRequired
  ).length
  const rejected = approvals.filter((item) => item.rejected).length
  const replicas = resources.reduce(
    (sum, item) => sum + (item.status?.replicas || 0),
    0
  )
  const available = resources.reduce(
    (sum, item) => sum + (item.status?.availableReplicas || 0),
    0
  )
  const updates = stats.reduce((sum, item) => sum + item.updates, 0)
  const shown = useMemo(
    () =>
      resources.filter((item) =>
        [
          item.identifier,
          item.namespace,
          item.policy,
          item.provider,
          ...(item.images || []),
        ].some((value) => value?.toLowerCase().includes(filter.toLowerCase()))
      ),
    [resources, filter]
  )
  async function mutate(action: () => Promise<unknown>, success: string) {
    try {
      await action()
      toast.success(success)
      await load()
    } catch (error) {
      toast.error("Update failed", { description: String(error) })
    }
  }
  function setPolicy(resource: Resource, policy: string) {
    return mutate(
      () =>
        api.setPolicy({
          identifier: resource.identifier,
          provider: resource.provider,
          policy,
        }),
      `${resource.kind} ${resource.name} policy set to ${policy}`
    )
  }
  function setApproval(resource: Resource, increase: boolean) {
    const current = Number(resource.annotations?.["keel.sh/approvals"] || 0)
    return mutate(
      () =>
        api.setApprovalCount({
          identifier: resource.identifier,
          provider: resource.provider,
          votesRequired: increase ? current + 1 : Math.max(0, current - 1),
        }),
      `${resource.kind} ${resource.name} approvals updated`
    )
  }
  return (
    <div className="grid gap-6">
      <PageHeading
        title="Cluster overview"
        description="Monitor workload health, update activity, policies, and approvals across your Kubernetes cluster."
        eyebrow="Dashboard"
      />
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          title="Cluster Resources"
          value={resources.length}
          footer={
            <>
              Managed by Keel:{" "}
              {resources.filter((item) => item.policy !== "nil policy").length}
            </>
          }
        />
        <StatCard
          title="Total pods in cluster"
          value={replicas}
          footer={
            <>
              Healthy: {available} · Unavailable: {replicas - available} ·{" "}
              {replicas ? Math.round((available * 100) / replicas) : 0}% up
            </>
          }
        />
        <StatCard
          title="Updates"
          value={updates}
          footer={<>Average {Math.round(updates / 4)} updates per week</>}
        />
        <StatCard
          title="Pending Approvals"
          value={pending.length}
          footer={
            <>
              Approved: {approved} · Rejected: {rejected}
            </>
          }
        />
      </div>
      <Card className="border-white/10 bg-card/60 shadow-none">
        <CardHeader className="gap-4 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle>Kubernetes Cluster Resources</CardTitle>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => void load()}>
              <RefreshCw />
              Refresh
            </Button>
            <Input
              aria-label="Search resources"
              placeholder="Search resources"
              value={filter}
              onChange={(event) => setFilter(event.target.value)}
              className="max-w-64"
            />
          </div>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Namespace</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Pods</TableHead>
                <TableHead>Policy</TableHead>
                <TableHead>Approvals</TableHead>
                <TableHead>Images</TableHead>
                <TableHead>Keel Labels & Annotations</TableHead>
                <TableHead>Controls</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {shown.map((resource) => (
                <ResourceRow
                  key={resource.identifier}
                  resource={resource}
                  setPolicy={setPolicy}
                  setApproval={setApproval}
                  mutate={mutate}
                  openPolicy={setPolicyDialog}
                />
              ))}
              {!shown.length && (
                <EmptyRow columns={8}>
                  {loading ? "Loading…" : "No resources found."}
                </EmptyRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      <Dialog
        open={Boolean(policyDialog)}
        onOpenChange={(open) => {
          if (!open) {
            setPolicyDialog(null)
            setPolicyInput("")
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              Set policy for {policyDialog?.resource.identifier}
            </DialogTitle>
            <DialogDescription>
              {policyDialog?.kind === "glob"
                ? "Use wildcards to match tags, for example build-*."
                : "Use RE2 regular expressions to match versions."}
            </DialogDescription>
          </DialogHeader>
          <div className="flex">
            <span className="rounded-l-md border border-r-0 bg-muted px-3 py-2 text-sm">
              {policyDialog?.kind}:
            </span>
            <Input
              autoFocus
              className="rounded-l-none"
              placeholder={
                policyDialog?.kind === "glob" ? "build-*" : "^([a-zA-Z]+)$"
              }
              value={policyInput}
              onChange={(event) => setPolicyInput(event.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPolicyDialog(null)}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                if (policyDialog)
                  void setPolicy(
                    policyDialog.resource,
                    `${policyDialog.kind}:${policyInput}`
                  )
                setPolicyDialog(null)
                setPolicyInput("")
              }}
              disabled={!policyInput}
            >
              Set policy
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function ResourceRow({
  resource,
  setPolicy,
  setApproval,
  mutate,
  openPolicy,
}: {
  resource: Resource
  setPolicy: (resource: Resource, policy: string) => Promise<void>
  setApproval: (resource: Resource, increase: boolean) => Promise<void>
  mutate: (action: () => Promise<unknown>, success: string) => Promise<void>
  openPolicy: (value: PolicyDialog) => void
}) {
  const managed = resource.policy !== "nil policy"
  const available = resource.status?.availableReplicas || 0
  const replicas = resource.status?.replicas || 0
  const opts = {
    ...Object.fromEntries(
      Object.entries(resource.labels || {}).filter(([key]) =>
        key.startsWith("keel.sh/")
      )
    ),
    ...Object.fromEntries(
      Object.entries(resource.annotations || {}).filter(([key]) =>
        key.startsWith("keel.sh/")
      )
    ),
  }
  const polling =
    (resource.annotations?.["keel.sh/trigger"] ||
      resource.labels?.["keel.sh/trigger"]) === "poll"
  return (
    <TableRow>
      <TableCell>{resource.namespace}</TableCell>
      <TableCell className="font-medium">
        {resource.kind}/{resource.name}
      </TableCell>
      <TableCell>
        <Badge variant={available === replicas ? "default" : "secondary"}>
          {available}/{replicas}
        </Badge>
      </TableCell>
      <TableCell>
        <Badge variant={managed ? "default" : "outline"}>
          {managed ? resource.policy : "none"}
        </Badge>
      </TableCell>
      <TableCell>
        {resource.annotations?.["keel.sh/approvals"] || "-"}
      </TableCell>
      <TableCell>
        <div className="flex max-w-64 flex-wrap gap-1">
          {(resource.images || []).map((image) => (
            <Badge variant="outline" key={image}>
              {image}
            </Badge>
          ))}
        </div>
      </TableCell>
      <TableCell>
        <div className="flex max-w-72 flex-wrap gap-1">
          {Object.entries(opts).map(([key, value]) => (
            <Badge variant="secondary" key={key}>
              {key}: {value}
            </Badge>
          ))}
        </div>
      </TableCell>
      <TableCell>
        <div className="flex min-w-72 items-center gap-1">
          <Button
            size="sm"
            disabled={!managed}
            onClick={() => void setPolicy(resource, "never")}
          >
            <Pause />
            Pause
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button size="sm" />}>
              <span>Policy</span>
              <ChevronDown />
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              {["patch", "minor", "major", "all", "force"].map((policy) => (
                <DropdownMenuItem
                  key={policy}
                  onClick={() => void setPolicy(resource, policy)}
                >
                  {policy}
                </DropdownMenuItem>
              ))}
              <DropdownMenuItem
                onClick={() => openPolicy({ resource, kind: "glob" })}
              >
                glob
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => openPolicy({ resource, kind: "regexp" })}
              >
                regexp
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button
            size="icon-sm"
            onClick={() => void setApproval(resource, true)}
            title="Increase required approvals"
          >
            <Plus />
            <span className="sr-only">Increase required approvals</span>
          </Button>
          <Button
            size="icon-sm"
            onClick={() => void setApproval(resource, false)}
            title="Decrease required approvals"
          >
            <Minus />
            <span className="sr-only">Decrease required approvals</span>
          </Button>
          <Switch
            aria-label={`Polling for ${resource.name}`}
            checked={polling}
            disabled={!managed}
            onCheckedChange={(checked) =>
              void mutate(
                () =>
                  api.setTracking({
                    identifier: resource.identifier,
                    provider: resource.provider,
                    trigger: checked ? "poll" : "default",
                  }),
                `${resource.kind} ${resource.name} tracking updated`
              )
            }
          />
        </div>
      </TableCell>
    </TableRow>
  )
}

import { useCallback, useEffect, useMemo, useState } from "react"
import {
  EllipsisVertical,
  Pause,
  Play,
  Radar,
  RefreshCw,
  ShieldCheck,
  SlidersHorizontal,
} from "lucide-react"
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
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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

type ActionKind = "approvals" | "policy" | "pause"
type ActionDialog = { resource: Resource; kind: ActionKind } | null

const policyOptions = ["patch", "minor", "major", "all", "force"] as const
const policyPickerOptions = [
  ...policyOptions,
  "glob",
  "regexp",
  "none",
] as const
const policyExamples: Record<string, string> = {
  patch: "Patch updates only. For example, 1.2.3 → 1.2.4.",
  minor: "Minor and patch updates. For example, 1.2.3 → 1.3.0.",
  major: "Major, minor, and patch updates. For example, 1.2.3 → 2.0.0.",
  all: "Any newer semantic version, including prereleases.",
  force: "Update even when tags are not semantic versions.",
  glob: "Match wildcard tag patterns, such as release-*.",
  regexp: String.raw`Match tags with RE2, such as ^v\d+\.\d+\.\d+$.`,
  none: "Clear the policy so Keel no longer applies updates to this workload.",
}
const patternPresets = {
  glob: [
    {
      pattern: "release-*",
      description:
        "Matches tags beginning with release-, such as release-2026.08.",
    },
    {
      pattern: "v1.*",
      description: "Matches only v1 tags, such as v1.12.0.",
    },
    {
      pattern: "*-alpine",
      description: "Matches tags ending in -alpine.",
    },
  ],
  regexp: [
    {
      pattern: String.raw`^v\d+\.\d+\.\d+$`,
      description: "Matches exact v-prefixed versions, such as v2.4.1.",
    },
    {
      pattern: String.raw`^release-\d{8}$`,
      description: "Matches dated tags, such as release-20260804.",
    },
    {
      pattern: String.raw`^(main|stable)-\d+$`,
      description: "Matches numbered main or stable builds.",
    },
  ],
}

export function DashboardPage() {
  const [resources, setResources] = useState<Resource[]>([])
  const [approvals, setApprovals] = useState<Approval[]>([])
  const [stats, setStats] = useState<Stats[]>([])
  const [filter, setFilter] = useState("")
  const [loading, setLoading] = useState(true)
  const [actionDialog, setActionDialog] = useState<ActionDialog>(null)
  const [approvalInput, setApprovalInput] = useState("0")
  const [policyChoice, setPolicyChoice] = useState("patch")
  const [policyInput, setPolicyInput] = useState("")
  const [resumePolicy, setResumePolicy] = useState("patch")
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
      policy
        ? `${resource.kind} ${resource.name} policy set to ${policy}`
        : `${resource.kind} ${resource.name} policy cleared`
    )
  }
  function setApproval(resource: Resource, votesRequired: number) {
    return mutate(
      () =>
        api.setApprovalCount({
          identifier: resource.identifier,
          provider: resource.provider,
          votesRequired,
        }),
      `${resource.kind} ${resource.name} now requires ${votesRequired} approval${votesRequired === 1 ? "" : "s"}`
    )
  }

  function openAction(resource: Resource, kind: ActionKind) {
    setActionDialog({ resource, kind })
    if (kind === "approvals") {
      setApprovalInput(resource.annotations?.["keel.sh/approvals"] || "0")
    }
    if (kind === "policy") {
      const [currentPolicy, ...pattern] = resource.policy.split(":")
      const supported = policyPickerOptions as readonly string[]
      const selectedPolicy =
        currentPolicy === "nil policy" ? "none" : currentPolicy
      setPolicyChoice(
        supported.includes(selectedPolicy) ? selectedPolicy : "patch"
      )
      setPolicyInput(pattern.join(":"))
    }
    if (kind === "pause") {
      setResumePolicy(
        resource.policy !== "never" &&
          policyOptions.includes(
            resource.policy as (typeof policyOptions)[number]
          )
          ? resource.policy
          : "patch"
      )
    }
  }

  function closeAction() {
    setActionDialog(null)
    setPolicyInput("")
  }

  const requestedApprovals = Number(approvalInput)
  const approvalsValid =
    Number.isInteger(requestedApprovals) && requestedApprovals >= 0
  return (
    <div className="grid gap-6">
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
      <Card className="border-border bg-card/60 shadow-none">
        <CardHeader className="gap-4 xl:grid-cols-[1fr_auto] xl:items-center">
          <CardTitle>Kubernetes Cluster Resources</CardTitle>
          <div className="grid w-full grid-cols-[auto_minmax(0,1fr)] gap-2 sm:flex xl:w-auto">
            <Button variant="outline" onClick={() => void load()}>
              <RefreshCw />
              Refresh
            </Button>
            <Input
              aria-label="Search resources"
              placeholder="Search resources"
              value={filter}
              onChange={(event) => setFilter(event.target.value)}
              className="min-w-0 sm:max-w-64"
            />
          </div>
        </CardHeader>
        <CardContent>
          <Table className="md:table-fixed 2xl:table-auto">
            <TableHeader className="hidden md:table-header-group">
              <TableRow>
                <TableHead className="hidden lg:table-cell lg:w-28 xl:w-auto">
                  Namespace
                </TableHead>
                <TableHead>Name</TableHead>
                <TableHead className="w-16">Pods</TableHead>
                <TableHead className="w-20">Policy</TableHead>
                <TableHead className="hidden xl:table-cell xl:w-24 2xl:w-auto">
                  Approvals
                </TableHead>
                <TableHead className="hidden xl:table-cell xl:w-72 2xl:w-auto">
                  Images
                </TableHead>
                <TableHead className="hidden 2xl:table-cell">
                  Keel metadata
                </TableHead>
                <TableHead className="w-14 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody className="grid gap-3 md:table-row-group">
              {shown.map((resource) => (
                <ResourceRow
                  key={resource.identifier}
                  resource={resource}
                  mutate={mutate}
                  openAction={openAction}
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
      <Dialog open={Boolean(actionDialog)} onOpenChange={closeAction}>
        {actionDialog && (
          <DialogContent className="max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>
                {actionDialog.kind === "approvals" &&
                  "Adjust required approvals"}
                {actionDialog.kind === "policy" && "Change update policy"}
                {actionDialog.kind === "pause" &&
                  (actionDialog.resource.policy === "never"
                    ? "Resume updates"
                    : "Pause updates")}
              </DialogTitle>
              <DialogDescription>
                {actionDialog.kind === "approvals" &&
                  "Require a specific number of votes before Keel applies an available update. Set this to zero to disable approval gating."}
                {actionDialog.kind === "policy" &&
                  "Choose which image version changes Keel may apply to this workload."}
                {actionDialog.kind === "pause" &&
                  (actionDialog.resource.policy === "never"
                    ? "Choose the update policy Keel should use when automation resumes."
                    : "Keel will stop applying image updates until this workload is resumed.")}
              </DialogDescription>
            </DialogHeader>

            <p className="truncate rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
              {actionDialog.resource.identifier}
            </p>

            {actionDialog.kind === "approvals" && (
              <div className="grid gap-2">
                <Label htmlFor="required-approvals">Required approvals</Label>
                <Input
                  id="required-approvals"
                  autoFocus
                  type="number"
                  min="0"
                  step="1"
                  value={approvalInput}
                  onChange={(event) => setApprovalInput(event.target.value)}
                />
                {!approvalsValid && (
                  <p className="text-xs text-destructive">
                    Enter a whole number of zero or greater.
                  </p>
                )}
              </div>
            )}

            {actionDialog.kind === "policy" && (
              <PolicyPicker
                value={policyChoice}
                onChange={setPolicyChoice}
                pattern={policyInput}
                onPatternChange={setPolicyInput}
              />
            )}

            {actionDialog.kind === "pause" &&
              actionDialog.resource.policy === "never" && (
                <div className="grid gap-2">
                  <Label>Resume with policy</Label>
                  <PolicyButtons
                    value={resumePolicy}
                    onChange={setResumePolicy}
                  />
                  <p className="text-xs text-muted-foreground">
                    You can configure glob or regular-expression policies from
                    Change update policy after resuming.
                  </p>
                </div>
              )}

            {actionDialog.kind === "pause" &&
              actionDialog.resource.policy !== "never" && (
                <div className="rounded-md border bg-muted/30 p-3 text-sm">
                  Current policy:{" "}
                  <strong>{actionDialog.resource.policy}</strong>
                </div>
              )}

            <DialogFooter>
              <Button variant="outline" onClick={closeAction}>
                Cancel
              </Button>
              {actionDialog.kind === "approvals" && (
                <Button
                  disabled={!approvalsValid}
                  onClick={() => {
                    void setApproval(actionDialog.resource, requestedApprovals)
                    closeAction()
                  }}
                >
                  Save approvals
                </Button>
              )}
              {actionDialog.kind === "policy" && (
                <Button
                  disabled={
                    (policyChoice === "glob" || policyChoice === "regexp") &&
                    !policyInput.trim()
                  }
                  onClick={() => {
                    const nextPolicy =
                      policyChoice === "none"
                        ? ""
                        : policyChoice === "glob" || policyChoice === "regexp"
                          ? `${policyChoice}:${policyInput.trim()}`
                          : policyChoice
                    void setPolicy(actionDialog.resource, nextPolicy)
                    closeAction()
                  }}
                >
                  Save policy
                </Button>
              )}
              {actionDialog.kind === "pause" && (
                <Button
                  onClick={() => {
                    const paused = actionDialog.resource.policy === "never"
                    void setPolicy(
                      actionDialog.resource,
                      paused ? resumePolicy : "never"
                    )
                    closeAction()
                  }}
                >
                  {actionDialog.resource.policy === "never"
                    ? "Resume updates"
                    : "Pause updates"}
                </Button>
              )}
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  )
}

function ResourceRow({
  resource,
  mutate,
  openAction,
}: {
  resource: Resource
  mutate: (action: () => Promise<unknown>, success: string) => Promise<void>
  openAction: (resource: Resource, kind: ActionKind) => void
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
    <TableRow className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-2 rounded-lg border p-3 md:table-row md:rounded-none md:border-x-0 md:p-0">
      <TableCell className="hidden lg:table-cell">
        {resource.namespace}
      </TableCell>
      <TableCell className="min-w-0 overflow-hidden p-0 font-medium md:p-2">
        <span className="block truncate">
          {resource.kind}/{resource.name}
        </span>
        <span className="block truncate text-xs font-normal text-muted-foreground md:hidden">
          {resource.namespace}
        </span>
      </TableCell>
      <TableCell className="col-start-1 flex items-center gap-2 p-0 md:table-cell md:p-2">
        <span className="w-12 text-xs text-muted-foreground md:hidden">
          Pods
        </span>
        <Badge
          variant="outline"
          className={
            available === replicas ? undefined : "text-muted-foreground"
          }
        >
          {available}/{replicas}
        </Badge>
      </TableCell>
      <TableCell className="col-start-1 flex items-center gap-2 p-0 md:table-cell md:p-2">
        <span className="w-12 text-xs text-muted-foreground md:hidden">
          Policy
        </span>
        <Badge
          variant="outline"
          className={managed ? undefined : "text-muted-foreground"}
        >
          {managed ? resource.policy : "none"}
        </Badge>
      </TableCell>
      <TableCell className="hidden xl:table-cell">
        {resource.annotations?.["keel.sh/approvals"] || "-"}
      </TableCell>
      <TableCell className="hidden xl:table-cell">
        <div className="flex max-w-64 flex-wrap gap-1">
          {(resource.images || []).map((image) => (
            <Badge variant="outline" key={image}>
              {image}
            </Badge>
          ))}
        </div>
      </TableCell>
      <TableCell className="hidden 2xl:table-cell">
        <div className="flex max-w-72 flex-wrap gap-1">
          {Object.entries(opts).map(([key, value]) => (
            <Badge variant="secondary" key={key}>
              {key}: {value}
            </Badge>
          ))}
        </div>
      </TableCell>
      <TableCell className="col-start-2 row-span-3 row-start-1 p-0 text-right md:table-cell md:p-2">
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon"
                aria-label={`Actions for ${resource.kind} ${resource.name}`}
              />
            }
          >
            <EllipsisVertical />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuItem onClick={() => openAction(resource, "policy")}>
              <SlidersHorizontal />
              Change update policy
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => openAction(resource, "approvals")}>
              <ShieldCheck />
              Adjust required approvals
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={!managed}
              onClick={() => openAction(resource, "pause")}
            >
              {resource.policy === "never" ? <Play /> : <Pause />}
              {resource.policy === "never" ? "Resume updates" : "Pause updates"}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              disabled={!managed}
              onClick={() =>
                void mutate(
                  () =>
                    api.setTracking({
                      identifier: resource.identifier,
                      provider: resource.provider,
                      trigger: polling ? "default" : "poll",
                    }),
                  `${resource.kind} ${resource.name} registry polling ${polling ? "disabled" : "enabled"}`
                )
              }
            >
              <Radar />
              {polling ? "Disable registry polling" : "Enable registry polling"}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  )
}

function PolicyButtons({
  value,
  onChange,
  options = policyOptions,
  pattern,
  onPatternChange,
}: {
  value: string
  onChange: (value: string) => void
  options?: readonly string[]
  pattern?: string
  onPatternChange?: (value: string) => void
}) {
  return (
    <div className="grid gap-2">
      {options.map((policy) => (
        <div key={policy} className="grid gap-2">
          <div className="grid grid-cols-[5.5rem_minmax(0,1fr)] items-center gap-3">
            <Button
              type="button"
              aria-label={policy}
              variant={value === policy ? "default" : "outline"}
              onClick={() => onChange(policy)}
              className="w-full capitalize"
            >
              {policy}
            </Button>
            <p className="text-xs leading-5 text-muted-foreground">
              {policyExamples[policy]}
            </p>
          </div>
          {(policy === "glob" || policy === "regexp") &&
            value === policy &&
            pattern !== undefined &&
            onPatternChange && (
              <PolicyPatternEditor
                policy={policy}
                pattern={pattern}
                onPatternChange={onPatternChange}
              />
            )}
        </div>
      ))}
    </div>
  )
}

function PolicyPatternEditor({
  policy,
  pattern,
  onPatternChange,
}: {
  policy: "glob" | "regexp"
  pattern: string
  onPatternChange: (value: string) => void
}) {
  return (
    <div className="ml-[6.25rem] grid gap-2 border-l pl-3">
      <Label htmlFor="policy-pattern">
        {policy === "glob" ? "Wildcard pattern" : "RE2 expression"}
      </Label>
      <Input
        id="policy-pattern"
        autoFocus
        placeholder={policy === "glob" ? "build-*" : "^([a-zA-Z]+)$"}
        value={pattern}
        onChange={(event) => onPatternChange(event.target.value)}
      />
      <p className="text-xs text-muted-foreground">
        Choose an example or enter your own pattern.
      </p>
      <div className="grid gap-2">
        {patternPresets[policy].map((preset) => (
          <div
            className="flex flex-wrap items-center gap-2"
            key={preset.pattern}
          >
            <Button
              type="button"
              size="xs"
              variant="outline"
              aria-label={`Use ${preset.pattern}`}
              className="font-mono"
              onClick={() => onPatternChange(preset.pattern)}
            >
              {preset.pattern}
            </Button>
            <span className="min-w-48 flex-1 text-xs leading-5 text-muted-foreground">
              {preset.description}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

function PolicyPicker({
  value,
  onChange,
  pattern,
  onPatternChange,
}: {
  value: string
  onChange: (value: string) => void
  pattern: string
  onPatternChange: (value: string) => void
}) {
  return (
    <div className="grid gap-3">
      <Label>Policy</Label>
      <PolicyButtons
        value={value}
        onChange={onChange}
        options={policyPickerOptions}
        pattern={pattern}
        onPatternChange={onPatternChange}
      />
    </div>
  )
}

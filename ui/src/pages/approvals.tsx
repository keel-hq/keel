import { useCallback, useEffect, useMemo, useState } from "react"
import {
  Archive,
  Check,
  EllipsisVertical,
  RefreshCw,
  ThumbsDown,
  Trash2,
} from "lucide-react"
import { toast } from "sonner"
import { useAuth } from "@/auth"
import { api } from "@/lib/api"
import { deadline, formatDate } from "@/lib/format"
import type { Approval } from "@/types"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Progress } from "@/components/ui/progress"
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

type PendingAction = {
  approvals: Approval[]
  action: "approve" | "reject" | "archive" | "delete"
} | null
export function ApprovalsPage() {
  const { user } = useAuth()
  const [approvals, setApprovals] = useState<Approval[]>([])
  const [selected, setSelected] = useState<string[]>([])
  const [filter, setFilter] = useState("")
  const [pendingAction, setPendingAction] = useState<PendingAction>(null)
  const load = useCallback(async () => {
    try {
      setApprovals(await api.approvals())
    } catch (error) {
      toast.error("Could not load approvals", { description: String(error) })
    }
  }, [])
  useEffect(() => {
    void load()
  }, [load])
  const shown = useMemo(
    () =>
      approvals.filter((item) =>
        [item.identifier, item.provider, item.message, item.createdAt].some(
          (value) => value?.toLowerCase().includes(filter.toLowerCase())
        )
      ),
    [approvals, filter]
  )
  const isComplete = (item: Approval) =>
    item.archived || item.rejected || item.votesReceived >= item.votesRequired
  const pending = approvals.filter((item) => !isComplete(item))
  const approved = approvals.filter(
    (item) => !item.rejected && item.votesReceived >= item.votesRequired
  ).length
  const rejected = approvals.filter((item) => item.rejected).length
  async function execute() {
    if (!pendingAction) return
    try {
      await Promise.all(
        pendingAction.approvals.map((item) =>
          api.updateApproval({
            id: item.id,
            identifier: item.identifier,
            action: pendingAction.action,
            voter: user?.name || "admin-web-ui",
          })
        )
      )
      toast.success(`${pendingAction.action} completed`)
      setSelected([])
      await load()
    } catch (error) {
      toast.error(`${pendingAction.action} failed`, {
        description: String(error),
      })
    } finally {
      setPendingAction(null)
    }
  }
  const selectedApprovals = approvals.filter(
    (item) => selected.includes(item.id) && !isComplete(item)
  )
  return (
    <div className="grid gap-6">
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          title="Pending"
          value={pending.length}
          footer="Awaiting votes"
        />
        <StatCard title="Approved" value={approved} footer="Approved updates" />
        <StatCard title="Rejected" value={rejected} footer="Rejected updates" />
      </div>
      <Card className="border-border bg-card/60 shadow-none">
        <CardHeader className="gap-4">
          <CardTitle>Approvals</CardTitle>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" onClick={() => void load()}>
              <RefreshCw />
              Refresh
            </Button>
            <Button
              variant="outline"
              className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 hover:bg-emerald-500/20 hover:text-emerald-800 dark:text-emerald-400 dark:hover:text-emerald-300"
              disabled={!selectedApprovals.length}
              onClick={() =>
                setPendingAction({
                  approvals: selectedApprovals,
                  action: "approve",
                })
              }
            >
              <Check />
              Approve
            </Button>
            <Button
              variant="destructive"
              disabled={!selectedApprovals.length}
              onClick={() =>
                setPendingAction({
                  approvals: selectedApprovals,
                  action: "reject",
                })
              }
            >
              <ThumbsDown />
              Reject
            </Button>
            <Input
              aria-label="Search approvals"
              className="max-w-72"
              placeholder="Search approvals"
              value={filter}
              onChange={(event) => setFilter(event.target.value)}
            />
          </div>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  <span className="sr-only">Select</span>
                </TableHead>
                <TableHead>Last Activity</TableHead>
                <TableHead>Provider</TableHead>
                <TableHead>Identifier</TableHead>
                <TableHead>Votes</TableHead>
                <TableHead>Delta</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Expires In</TableHead>
                <TableHead>Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {shown.map((item) => {
                const complete = isComplete(item)
                const progress = item.votesRequired
                  ? Math.min(
                      100,
                      (item.votesReceived * 100) / item.votesRequired
                    )
                  : 100
                const status = item.archived
                  ? "Archived"
                  : item.rejected
                    ? "Rejected"
                    : complete
                      ? "Complete"
                      : "Collecting…"
                return (
                  <TableRow key={item.id}>
                    <TableCell>
                      <Checkbox
                        aria-label={`Select ${item.identifier}`}
                        disabled={complete}
                        checked={selected.includes(item.id)}
                        onCheckedChange={(checked) =>
                          setSelected((current) =>
                            checked
                              ? [...current, item.id]
                              : current.filter((id) => id !== item.id)
                          )
                        }
                      />
                    </TableCell>
                    <TableCell>{formatDate(item.updatedAt)}</TableCell>
                    <TableCell>{item.provider}</TableCell>
                    <TableCell className="font-medium">
                      {item.identifier}
                    </TableCell>
                    <TableCell>
                      {item.votesReceived}/{item.votesRequired}
                    </TableCell>
                    <TableCell>
                      {item.currentVersion} → {item.newVersion}
                    </TableCell>
                    <TableCell className="min-w-36">
                      <span>{status}</span>
                      <Progress value={progress} className="mt-1" />
                    </TableCell>
                    <TableCell title={item.deadline}>
                      {complete ? "-" : deadline(item.deadline)}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button
                          size="icon-sm"
                          variant="outline"
                          className="border-emerald-500/30 bg-emerald-500/15 text-emerald-700 hover:bg-emerald-500/25 hover:text-emerald-800 dark:text-emerald-400 dark:hover:text-emerald-300"
                          disabled={complete}
                          onClick={() =>
                            setPendingAction({
                              approvals: [item],
                              action: "approve",
                            })
                          }
                          title="Approve"
                        >
                          <Check />
                        </Button>
                        <Button
                          size="icon-sm"
                          variant="destructive"
                          disabled={complete}
                          onClick={() =>
                            setPendingAction({
                              approvals: [item],
                              action: "reject",
                            })
                          }
                          title="Reject"
                        >
                          <ThumbsDown />
                        </Button>
                        <DropdownMenu>
                          <DropdownMenuTrigger
                            render={
                              <Button
                                size="icon-sm"
                                variant="ghost"
                                aria-label={`More actions for ${item.identifier}`}
                              />
                            }
                          >
                            <EllipsisVertical />
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end" className="w-36">
                            <DropdownMenuItem
                              disabled={item.archived}
                              onClick={() =>
                                setPendingAction({
                                  approvals: [item],
                                  action: "archive",
                                })
                              }
                            >
                              <Archive />
                              Archive
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              variant="destructive"
                              onClick={() =>
                                setPendingAction({
                                  approvals: [item],
                                  action: "delete",
                                })
                              }
                            >
                              <Trash2 />
                              Delete
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })}
              {!shown.length && (
                <EmptyRow columns={9}>No approvals found.</EmptyRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      <AlertDialog
        open={Boolean(pendingAction)}
        onOpenChange={(open) => {
          if (!open) setPendingAction(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Confirm {pendingAction?.action}</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to {pendingAction?.action}{" "}
              {pendingAction?.approvals.length === 1
                ? pendingAction.approvals[0].identifier
                : `${pendingAction?.approvals.length} selected updates`}
              ?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => void execute()}>
              Confirm
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

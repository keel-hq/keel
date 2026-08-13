import { useCallback, useEffect, useState } from "react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { formatDate } from "@/lib/format"
import type { AuditLog } from "@/types"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
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

export function AuditLogsPage() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  const load = useCallback(async () => {
    setLoading(true)
    try {
      setLogs((await api.audit()).data || [])
    } catch (error) {
      toast.error("Could not load audit logs", { description: String(error) })
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    void load()
  }, [load])
  return (
    <div className="grid gap-6">
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          title="Last Event"
          value={logs[0] ? formatDate(logs[0].createdAt) : "-"}
          footer="Most recent audit entry"
        />
        <StatCard
          title="Audit Entries"
          value={logs.length}
          footer="Entries returned"
        />
        <StatCard
          title="Registries"
          value="-"
          footer="Registry summary unavailable"
        />
      </div>
      <Card className="border-border bg-card/60 shadow-none">
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle>Audit Logs</CardTitle>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Time</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Resource Kind</TableHead>
                <TableHead>Identifier</TableHead>
                <TableHead>Metadata</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell>{formatDate(log.createdAt)}</TableCell>
                  <TableCell>{log.action}</TableCell>
                  <TableCell>{log.resourceKind}</TableCell>
                  <TableCell className="font-medium">
                    {log.identifier}
                  </TableCell>
                  <TableCell>
                    <div className="flex max-w-xl flex-wrap gap-1">
                      {Object.entries(log.metadata || {}).map(
                        ([key, value]) => (
                          <Badge variant="secondary" key={key}>
                            {key}: {value}
                          </Badge>
                        )
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
              {!logs.length && (
                <EmptyRow columns={5}>
                  {loading ? "Loading…" : "No audit entries found."}
                </EmptyRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}

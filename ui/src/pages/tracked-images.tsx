import { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { TrackedImage } from "@/types"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
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

export function TrackedImagesPage() {
  const [images, setImages] = useState<TrackedImage[]>([])
  const [filter, setFilter] = useState("")
  const load = useCallback(async () => {
    try {
      setImages(await api.tracked())
    } catch (error) {
      toast.error("Could not load tracked images", {
        description: String(error),
      })
    }
  }, [])
  useEffect(() => {
    void load()
  }, [load])
  const shown = useMemo(
    () =>
      images.filter((image) =>
        Object.values(image).some((value) =>
          String(value).toLowerCase().includes(filter.toLowerCase())
        )
      ),
    [images, filter]
  )
  return (
    <div className="grid gap-6">
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          title="Namespaces"
          value={new Set(images.map((item) => item.namespace)).size}
          footer="Namespaces with tracked images"
        />
        <StatCard
          title="Total Images Tracked"
          value={images.length}
          footer="Images observed by Keel"
        />
        <StatCard
          title="Registries"
          value={new Set(images.map((item) => item.registry)).size}
          footer="Distinct registries"
        />
      </div>
      <Card className="border-border bg-card/60 shadow-none">
        <CardHeader className="gap-4 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle>Tracked Images</CardTitle>
          <div className="flex gap-2">
            <Input
              aria-label="Search tracked images"
              placeholder="Search tracked images"
              value={filter}
              onChange={(event) => setFilter(event.target.value)}
            />
          </div>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Image Name</TableHead>
                <TableHead>Provider</TableHead>
                <TableHead>Namespace</TableHead>
                <TableHead>Policy</TableHead>
                <TableHead>Trigger</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {shown.map((image, index) => (
                <TableRow key={`${image.image}-${index}`}>
                  <TableCell className="font-medium">{image.image}</TableCell>
                  <TableCell>{image.provider}</TableCell>
                  <TableCell>{image.namespace}</TableCell>
                  <TableCell>{image.policy}</TableCell>
                  <TableCell>
                    {image.trigger === "poll"
                      ? `poll - ${image.pollSchedule}`
                      : "webhook/GCR"}
                  </TableCell>
                </TableRow>
              ))}
              {!shown.length && (
                <EmptyRow columns={5}>No tracked images found.</EmptyRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}

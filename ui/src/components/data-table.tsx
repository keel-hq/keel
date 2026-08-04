import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
export function EmptyRow({
  columns,
  children = "No data available.",
}: {
  columns: number
  children?: React.ReactNode
}) {
  return (
    <TableRow>
      <TableCell
        colSpan={columns}
        className="h-24 text-center text-muted-foreground"
      >
        {children}
      </TableCell>
    </TableRow>
  )
}
export { Table, TableBody, TableCell, TableHead, TableHeader, TableRow }

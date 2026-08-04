import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
export function StatCard({
  title,
  value,
  footer,
}: {
  title: string
  value: string | number
  footer: React.ReactNode
}) {
  return (
    <Card className="gap-0 overflow-hidden border-white/10 bg-card/70 py-0 shadow-none">
      <CardHeader className="px-5 pt-5 pb-3">
        <CardTitle className="text-xs font-medium tracking-[0.14em] text-muted-foreground uppercase">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="px-5 pb-5 text-3xl font-medium tracking-tight tabular-nums">
        {value}
      </CardContent>
      <CardFooter className="border-t border-white/8 bg-white/[.015] px-5 py-3 text-xs text-muted-foreground">
        {footer}
      </CardFooter>
    </Card>
  )
}

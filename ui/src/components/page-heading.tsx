export function PageHeading({
  title,
  description,
  eyebrow,
}: {
  title: string
  description: string
  eyebrow?: string
}) {
  return (
    <div className="flex flex-col gap-1 border-b border-border pb-6">
      <span className="text-xs font-medium tracking-[0.18em] text-muted-foreground uppercase">
        {eyebrow || "Overview"}
      </span>
      <h1 className="mt-2 text-2xl font-semibold tracking-tight md:text-3xl">
        {title}
      </h1>
      <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
        {description}
      </p>
    </div>
  )
}

import { cn } from "@/lib/utils"

export function BrandLogo({ className }: { className?: string }) {
  return (
    <img
      src="/img/keel-logo.png"
      alt="Keel"
      className={cn("object-contain brightness-0 dark:invert", className)}
    />
  )
}

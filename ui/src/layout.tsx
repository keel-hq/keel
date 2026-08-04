import { useState } from "react"
import { Link, NavLink, Outlet } from "react-router-dom"
import {
  CheckSquare,
  ChevronRight,
  GitBranch,
  LayoutDashboard,
  LogOut,
  Menu,
  ScrollText,
} from "lucide-react"
import { useAuth } from "@/auth"
import { BrandLogo } from "@/components/brand-logo"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import { cn } from "@/lib/utils"

const links = [
  ["/dashboard", "Dashboard", LayoutDashboard],
  ["/tracked-images", "Tracked Images", GitBranch],
  ["/approvals", "Approvals", CheckSquare],
  ["/audit-logs", "Audit Logs", ScrollText],
] as const
function Navigation({ close }: { close?: () => void }) {
  return (
    <nav className="grid gap-1 px-3">
      {links.map(([to, label, Icon]) => (
        <NavLink
          key={to}
          to={to}
          onClick={close}
          className={({ isActive }) =>
            cn(
              "group flex items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-foreground",
              isActive && "bg-white text-black hover:bg-white hover:text-black"
            )
          }
        >
          <Icon className="size-4" />
          {label}
          <ChevronRight className="ml-auto size-3 opacity-0 transition-opacity group-hover:opacity-60" />
        </NavLink>
      ))}
    </nav>
  )
}
export function AppLayout() {
  const { user, logout } = useAuth()
  const [open, setOpen] = useState(false)
  return (
    <div className="min-h-svh bg-black">
      <aside className="fixed inset-y-0 left-0 hidden w-60 border-r border-white/10 bg-sidebar md:block">
        <Link
          to="/dashboard"
          className="flex h-16 items-center gap-3 border-b border-white/10 px-5"
        >
          <BrandLogo className="size-8" />
          <span className="font-semibold tracking-tight">Keel</span>
          <span className="ml-auto rounded-full border border-white/10 px-2 py-0.5 text-[10px] tracking-wider text-muted-foreground uppercase">
            Console
          </span>
        </Link>
        <div className="py-4">
          <p className="mb-2 px-6 text-[10px] font-medium tracking-[0.18em] text-muted-foreground uppercase">
            Workspace
          </p>
          <Navigation />
        </div>
        <div className="absolute inset-x-3 bottom-4 rounded-lg border border-white/10 bg-white/[.025] p-3">
          <div className="flex items-center gap-2 text-xs">
            <span className="size-1.5 rounded-full bg-emerald-400 shadow-[0_0_10px_rgba(52,211,153,.8)]" />
            <span>Keel connected</span>
          </div>
          <p className="mt-1.5 text-[11px] text-muted-foreground">
            Kubernetes automation
          </p>
        </div>
      </aside>
      <div className="md:pl-60">
        <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-white/10 bg-black/80 px-4 backdrop-blur-xl md:px-6">
          <div className="flex items-center gap-3">
            <Sheet open={open} onOpenChange={setOpen}>
              <SheetTrigger
                render={
                  <Button variant="ghost" size="icon" className="md:hidden" />
                }
              >
                <Menu />
                <span className="sr-only">Open navigation</span>
              </SheetTrigger>
              <SheetContent
                side="left"
                className="w-60 border-white/10 bg-black p-0"
              >
                <SheetTitle className="flex items-center gap-3 border-b border-white/10 p-5 text-left">
                  <BrandLogo className="size-8" />
                  Keel
                </SheetTitle>
                <div className="py-4">
                  <Navigation close={() => setOpen(false)} />
                </div>
              </SheetContent>
            </Sheet>
            <div>
              <p className="text-sm font-medium">Keel Console</p>
              <p className="hidden text-xs text-muted-foreground sm:block">
                Manage automated workload updates
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3 text-sm">
            <span className="hidden text-muted-foreground sm:inline">
              {user?.name || "admin"}
            </span>
            <span className="grid size-7 place-items-center rounded-full border border-white/15 bg-white/5 text-[11px] font-medium">
              {(user?.name || "A").slice(0, 1).toUpperCase()}
            </span>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => void logout()}
              title="Log out"
            >
              <LogOut />
              <span className="sr-only">Log out</span>
            </Button>
          </div>
        </header>
        <main className="mx-auto max-w-[1600px] p-4 md:p-8">
          <Outlet />
        </main>
        <footer className="px-6 py-8 text-center text-xs text-muted-foreground">
          Keel · Kubernetes update automation
        </footer>
      </div>
    </div>
  )
}

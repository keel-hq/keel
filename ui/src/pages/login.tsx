import { useState, type FormEvent } from "react"
import { Navigate, useLocation, useNavigate } from "react-router-dom"
import { ArrowRight, GitFork, LockKeyhole, User } from "lucide-react"
import { toast } from "sonner"
import { useAuth } from "@/auth"
import { BrandLogo } from "@/components/brand-logo"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export function LoginPage() {
  const { user, login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState("")
  if (user) return <Navigate to="/dashboard" replace />
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError("")
    const data = new FormData(event.currentTarget)
    try {
      await login(String(data.get("username")), String(data.get("password")))
      toast.success("Login successful", { description: "Loading data…" })
      navigate(
        new URLSearchParams(location.search).get("redirect") || "/dashboard",
        { replace: true }
      )
    } catch (reason) {
      const message =
        reason instanceof Error ? reason.message : "Authentication failed"
      setError(message)
      toast.error("Authentication failed", { description: message })
    } finally {
      setSubmitting(false)
    }
  }
  return (
    <div className="relative flex min-h-svh overflow-hidden bg-black px-4">
      <div className="keel-grid pointer-events-none absolute inset-0" />
      <div className="pointer-events-none absolute top-0 left-1/2 h-[520px] w-[820px] -translate-x-1/2 rounded-full bg-white/[.045] blur-[120px]" />
      <header className="absolute inset-x-0 top-0 z-10 flex h-20 items-center justify-between px-6 lg:px-10">
        <a href="/" className="flex items-center gap-2.5 font-semibold">
          <BrandLogo className="size-9" />
          Keel
        </a>
        <a
          className="flex items-center gap-2 text-xs text-muted-foreground transition-colors hover:text-white"
          href="https://github.com/keel-hq/keel"
          target="_blank"
          rel="noreferrer"
        >
          <GitFork className="size-4" />
          View on GitHub
        </a>
      </header>
      <main className="relative z-10 m-auto w-full max-w-[420px] py-28">
        <div className="mb-8">
          <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[.035] px-3 py-1 text-[11px] text-muted-foreground">
            <span className="size-1.5 rounded-full bg-emerald-400" />
            Keel administration console
          </div>
          <h1 className="text-3xl font-semibold tracking-[-.035em] sm:text-4xl">
            Sign in to Keel
          </h1>
          <p className="mt-3 text-sm leading-6 text-muted-foreground">
            Manage automated Kubernetes workload updates from one secure control
            plane.
          </p>
        </div>
        <Card className="border-white/10 bg-white/[.025] py-0 shadow-2xl shadow-black">
          <CardContent className="p-6 sm:p-7">
            <form className="grid gap-5" onSubmit={submit}>
              <div className="grid gap-2">
                <Label htmlFor="username" className="text-xs">
                  Username
                </Label>
                <div className="relative">
                  <User className="absolute top-3 left-3 size-4 text-muted-foreground" />
                  <Input
                    id="username"
                    name="username"
                    autoComplete="username"
                    className="h-10 border-white/10 bg-black/60 pl-9 placeholder:text-neutral-600 focus-visible:border-white/25"
                    placeholder="admin"
                    required
                  />
                </div>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="password" className="text-xs">
                  Password
                </Label>
                <div className="relative">
                  <LockKeyhole className="absolute top-3 left-3 size-4 text-muted-foreground" />
                  <Input
                    id="password"
                    name="password"
                    type="password"
                    autoComplete="current-password"
                    className="h-10 border-white/10 bg-black/60 pl-9 placeholder:text-neutral-600 focus-visible:border-white/25"
                    placeholder="Enter your password"
                    required
                  />
                </div>
              </div>
              {error && (
                <p
                  role="alert"
                  className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive"
                >
                  {error}
                </p>
              )}
              <Button
                type="submit"
                size="lg"
                className="mt-1 w-full"
                disabled={submitting}
              >
                {submitting ? (
                  "Signing in…"
                ) : (
                  <>
                    Continue <ArrowRight className="ml-auto" />
                  </>
                )}
              </Button>
            </form>
          </CardContent>
        </Card>
        <p className="mt-5 text-center text-[11px] leading-5 text-neutral-600">
          Authentication is handled by your Keel deployment.
          <br />
          Credentials never leave this server.
        </p>
      </main>
    </div>
  )
}

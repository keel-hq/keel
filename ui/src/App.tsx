import { Navigate, Outlet, Route, Routes, useLocation } from "react-router-dom"
import { useAuth } from "@/auth"
import { AppLayout } from "@/layout"
import { ApprovalsPage } from "@/pages/approvals"
import { AuditLogsPage } from "@/pages/audit-logs"
import { DashboardPage } from "@/pages/dashboard"
import { LoginPage } from "@/pages/login"
import { TrackedImagesPage } from "@/pages/tracked-images"
function RequireAuth() {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading)
    return (
      <div className="grid min-h-svh place-items-center text-sm text-muted-foreground">
        Loading Keel…
      </div>
    )
  return user ? (
    <Outlet />
  ) : (
    <Navigate
      to={`/user/login?redirect=${encodeURIComponent(location.pathname + location.search)}`}
      replace
    />
  )
}
export default function App() {
  return (
    <Routes>
      <Route path="/user/login" element={<LoginPage />} />
      <Route element={<RequireAuth />}>
        <Route element={<AppLayout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="tracked-images" element={<TrackedImagesPage />} />
          <Route path="approvals" element={<ApprovalsPage />} />
          <Route path="audit-logs" element={<AuditLogsPage />} />
        </Route>
      </Route>
      <Route
        path="*"
        element={
          <div className="grid min-h-svh place-items-center text-center">
            <div>
              <p className="text-6xl font-semibold">404</p>
              <p className="mt-2 text-muted-foreground">Page not found</p>
              <a className="mt-6 inline-block underline" href="/">
                Return to Keel
              </a>
            </div>
          </div>
        }
      />
    </Routes>
  )
}

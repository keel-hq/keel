import { render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { AuthProvider, useAuth } from "@/auth"
import { tokenStore } from "@/lib/api"

function AuthState() {
  const { user, loading } = useAuth()
  return <div>{loading ? "loading" : user?.name || "local-login-required"}</div>
}

describe("AuthProvider", () => {
  beforeEach(() => tokenStore.clear())
  afterEach(() => vi.unstubAllGlobals())

  it("discovers an external-proxy session without a Keel token or login", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            id: "1",
            name: "alice",
            username: "",
            role_id: "admin",
            auth_mode: "external-proxy",
            logout_url: "/oauth2/sign_out?rd=/",
          }),
          { status: 200 }
        )
      )
    )

    render(
      <AuthProvider>
        <AuthState />
      </AuthProvider>
    )
    expect(screen.getByText("loading")).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText("alice")).toBeInTheDocument())
    expect(fetch).toHaveBeenCalledWith(
      "/v1/auth/user",
      expect.objectContaining({ headers: expect.any(Headers) })
    )
    const headers = new Headers(vi.mocked(fetch).mock.calls[0][1]?.headers)
    expect(headers.has("Authorization")).toBe(false)
  })
})

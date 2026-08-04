import { describe, expect, it, vi } from "vitest"
import { api, tokenStore } from "@/lib/api"

describe("Keel API client", () => {
  it("persists the bearer token returned by login", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ token: "signed-token" }), {
          status: 200,
        })
      )
    )
    await api.login("admin", "secret")
    expect(tokenStore.get()).toBe("signed-token")
    expect(fetch).toHaveBeenCalledWith(
      "/v1/auth/login",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ username: "admin", password: "secret" }),
      })
    )
  })

  it("sends the bearer token to existing API routes", async () => {
    tokenStore.set("signed-token")
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("[]", { status: 200 }))
    )
    await api.resources()
    const init = vi.mocked(fetch).mock.calls[0][1]
    expect(new Headers(init?.headers).get("Authorization")).toBe(
      "Bearer signed-token"
    )
    expect(vi.mocked(fetch).mock.calls[0][0]).toBe("/v1/resources")
  })

  it("clears authentication when the backend returns 401", async () => {
    tokenStore.set("expired")
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("Unauthorized", { status: 401 }))
    )
    await expect(api.user()).rejects.toMatchObject({ status: 401 })
    expect(tokenStore.get()).toBeNull()
  })
})

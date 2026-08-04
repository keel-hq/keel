import { describe, expect, it, vi } from "vitest"
// OpenAPI Generator emits JavaScript without declaration files.
// @ts-expect-error generated contract client is validated at runtime below
import { ApiClient, AuthApi, SystemApi, WebhooksApi } from "@/api/generated"

describe("generated Keel API client", () => {
  it("uses same-origin defaults and configurable authentication", () => {
    const client = new ApiClient("")
    expect(client.basePath).toBe("")
    expect(client.enableCookies).toBe(false)
    client.authentications.BasicAuth.username = "admin"
    client.authentications.BearerAuth.apiKey = "Bearer token"
    expect(client.authentications.BasicAuth.username).toBe("admin")
    expect(client.authentications.BearerAuth.apiKey).toBe("Bearer token")
  })

  it("retains stable routes and security declarations", async () => {
    const client = new ApiClient("")
    client.callApi = vi.fn(() =>
      Promise.resolve({ data: { name: "keel" }, response: { status: 200 } })
    )
    await new SystemApi(client).getVersion()
    await new AuthApi(client).getAuthInfoWithHttpInfo()
    await new WebhooksApi(client).receiveRegistryWebhookWithHttpInfo({
      events: [],
    })
    expect(client.callApi.mock.calls[0].slice(0, 2)).toEqual([
      "/version",
      "GET",
    ])
    expect(client.callApi.mock.calls[1][7]).toEqual(["BasicAuth", "BearerAuth"])
    expect(client.callApi.mock.calls[2][7]).toEqual([])
  })
})

import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { DashboardPage } from "@/pages/dashboard"

const apiMock = vi.hoisted(() => ({
  resources: vi.fn(),
  approvals: vi.fn(),
  stats: vi.fn(),
  setPolicy: vi.fn(),
  setApprovalCount: vi.fn(),
  setTracking: vi.fn(),
}))

vi.mock("@/lib/api", () => ({ api: apiMock }))

const resource = {
  provider: "kubernetes",
  identifier: "deployment/keel-demo/storefront",
  name: "storefront",
  namespace: "keel-demo",
  kind: "deployment",
  policy: "patch",
  images: ["nginx:1.27.5"],
  labels: {},
  annotations: { "keel.sh/approvals": "1" },
  status: { replicas: 2, availableReplicas: 2 },
}

describe("Dashboard resource actions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMock.resources.mockResolvedValue([resource])
    apiMock.approvals.mockResolvedValue([])
    apiMock.stats.mockResolvedValue([])
    apiMock.setPolicy.mockResolvedValue(undefined)
    apiMock.setApprovalCount.mockResolvedValue(undefined)
    apiMock.setTracking.mockResolvedValue(undefined)
  })

  it("adjusts required approvals from the actions menu", async () => {
    const user = userEvent.setup()
    render(<DashboardPage />)

    await openActions(user)
    await user.click(
      await screen.findByRole("menuitem", {
        name: "Adjust required approvals",
      })
    )

    const input = screen.getByLabelText("Required approvals")
    expect(input).toHaveValue(1)
    await user.clear(input)
    await user.type(input, "3")
    await user.click(screen.getByRole("button", { name: "Save approvals" }))

    await waitFor(() =>
      expect(apiMock.setApprovalCount).toHaveBeenCalledWith({
        identifier: resource.identifier,
        provider: resource.provider,
        votesRequired: 3,
      })
    )
  })

  it("changes the update policy from a dialog", async () => {
    const user = userEvent.setup()
    render(<DashboardPage />)

    await openActions(user)
    await user.click(
      await screen.findByRole("menuitem", { name: "Change update policy" })
    )
    await user.click(screen.getByRole("button", { name: "minor" }))
    await user.click(screen.getByRole("button", { name: "Save policy" }))

    await waitFor(() =>
      expect(apiMock.setPolicy).toHaveBeenCalledWith({
        identifier: resource.identifier,
        provider: resource.provider,
        policy: "minor",
      })
    )
  })

  it("populates glob and regexp patterns from explained examples", async () => {
    const user = userEvent.setup()
    render(<DashboardPage />)

    await openActions(user)
    await user.click(
      await screen.findByRole("menuitem", { name: "Change update policy" })
    )

    await user.click(screen.getByRole("button", { name: "glob" }))
    expect(screen.getAllByRole("button", { name: /^Use / })).toHaveLength(3)
    expect(
      screen.getByText(/Matches tags beginning with release-/)
    ).toBeVisible()
    await user.click(screen.getByRole("button", { name: "Use release-*" }))
    expect(screen.getByLabelText("Wildcard pattern")).toHaveValue("release-*")

    await user.click(screen.getByRole("button", { name: "regexp" }))
    expect(screen.getAllByRole("button", { name: /^Use / })).toHaveLength(3)
    expect(screen.getByText(/Matches exact v-prefixed versions/)).toBeVisible()
    await user.click(
      screen.getByRole("button", { name: String.raw`Use ^v\d+\.\d+\.\d+$` })
    )
    expect(screen.getByLabelText("RE2 expression")).toHaveValue(
      String.raw`^v\d+\.\d+\.\d+$`
    )
  })

  it("confirms pausing updates", async () => {
    const user = userEvent.setup()
    render(<DashboardPage />)

    await openActions(user)
    await user.click(
      await screen.findByRole("menuitem", { name: "Pause updates" })
    )
    await user.click(screen.getByRole("button", { name: "Pause updates" }))

    await waitFor(() =>
      expect(apiMock.setPolicy).toHaveBeenCalledWith({
        identifier: resource.identifier,
        provider: resource.provider,
        policy: "never",
      })
    )
  })

  it("requires a policy when resuming updates", async () => {
    apiMock.resources.mockResolvedValue([{ ...resource, policy: "never" }])
    const user = userEvent.setup()
    render(<DashboardPage />)

    await openActions(user)
    await user.click(
      await screen.findByRole("menuitem", { name: "Resume updates" })
    )
    await user.click(screen.getByRole("button", { name: "major" }))
    await user.click(screen.getByRole("button", { name: "Resume updates" }))

    await waitFor(() =>
      expect(apiMock.setPolicy).toHaveBeenCalledWith({
        identifier: resource.identifier,
        provider: resource.provider,
        policy: "major",
      })
    )
  })
})

async function openActions(user: ReturnType<typeof userEvent.setup>) {
  await screen.findByText("deployment/storefront")
  screen
    .getByRole("button", { name: "Actions for deployment storefront" })
    .focus()
  await user.keyboard("{Enter}")
}

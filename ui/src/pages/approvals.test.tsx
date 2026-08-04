import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { ApprovalsPage } from "@/pages/approvals"

const apiMock = vi.hoisted(() => ({
  approvals: vi.fn(),
  updateApproval: vi.fn(),
}))

vi.mock("@/lib/api", () => ({ api: apiMock }))

const approval = {
  id: "approval-1",
  provider: "kubernetes",
  identifier: "deployment/keel-demo/storefront:1.27.6",
  message: "A new image is available.",
  currentVersion: "1.27.5",
  newVersion: "1.27.6",
  votesRequired: 2,
  votesReceived: 0,
  rejected: false,
  archived: false,
  deadline: "2026-08-05T12:00:00Z",
  createdAt: "2026-08-04T12:00:00Z",
  updatedAt: "2026-08-04T12:00:00Z",
}

describe("Approval row actions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMock.approvals.mockResolvedValue([approval])
    apiMock.updateApproval.mockResolvedValue(undefined)
  })

  it("keeps archive and delete in the row overflow menu", async () => {
    const user = userEvent.setup()
    render(<ApprovalsPage />)

    await screen.findByText(approval.identifier)
    const moreActions = screen.getByRole("button", {
      name: `More actions for ${approval.identifier}`,
    })
    moreActions.focus()
    await user.keyboard("{Enter}")
    await user.click(await screen.findByRole("menuitem", { name: "Archive" }))
    await user.click(screen.getByRole("button", { name: "Confirm" }))

    await waitFor(() =>
      expect(apiMock.updateApproval).toHaveBeenCalledWith({
        id: approval.id,
        identifier: approval.identifier,
        action: "archive",
        voter: "admin-web-ui",
      })
    )
  })
})

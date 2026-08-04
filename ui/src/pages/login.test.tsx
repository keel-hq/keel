import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { describe, expect, it, vi } from "vitest"
import { LoginPage } from "@/pages/login"

vi.mock("@/auth", () => ({ useAuth: () => ({ user: null, login: vi.fn() }) }))

describe("LoginPage", () => {
  it("renders the existing username and password authentication flow", () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    )
    expect(
      screen.getByRole("heading", { name: "Sign in to Keel" })
    ).toBeInTheDocument()
    expect(screen.getByLabelText("Username")).toBeRequired()
    expect(screen.getByLabelText("Password")).toHaveAttribute(
      "type",
      "password"
    )
    expect(screen.getByRole("button", { name: /continue/i })).toBeEnabled()
  })
})

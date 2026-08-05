import { fireEvent, render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { describe, expect, it, vi } from "vitest"
import { LoginPage } from "@/pages/login"
import { ThemeProvider } from "@/components/theme-provider"

vi.mock("@/auth", () => ({
  useAuth: () => ({ user: null, loading: false, login: vi.fn() }),
}))

describe("LoginPage", () => {
  it("renders the existing username and password authentication flow", () => {
    render(
      <MemoryRouter>
        <ThemeProvider defaultTheme="dark">
          <LoginPage />
        </ThemeProvider>
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

  it("persists light and dark mode changes", () => {
    render(
      <MemoryRouter>
        <ThemeProvider defaultTheme="dark">
          <LoginPage />
        </ThemeProvider>
      </MemoryRouter>
    )

    fireEvent.click(
      screen.getByRole("button", { name: "Switch to light mode" })
    )

    expect(
      screen.getByRole("button", { name: "Switch to dark mode" })
    ).toBeInTheDocument()
    expect(localStorage.getItem("theme")).toBe("light")
  })
})

import "@testing-library/jest-dom/vitest"
import { cleanup } from "@testing-library/react"
import { afterEach, beforeEach } from "vitest"

beforeEach(() => localStorage.clear())
afterEach(() => cleanup())

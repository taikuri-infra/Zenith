import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/components/shell", () => ({ Shell: ({ children }: { children: React.ReactNode }) => <div>{children}</div> }));
vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true, ZENITH_MODE: "standalone" }));

import DocsPage from "../page";

describe("DocsPage (standalone)", () => {
  it("documents the compose edition only", () => {
    const { container } = render(<DocsPage />);
    const text = container.textContent || "";
    expect(text).not.toMatch(/freezenith\.com/);
    expect(text).not.toMatch(/Pro plan|sign up/i);
    expect(text).not.toMatch(/Enable Auth|Add Storage|Git Push/);
    expect(text).toMatch(/zen backup/);
    expect(text).toMatch(/zen upgrade/);
    expect(text).toMatch(/A record/);
    expect(screen.getByText(/Custom domain and HTTPS/)).toBeInTheDocument();
  });
});

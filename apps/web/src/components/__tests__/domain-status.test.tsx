import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DomainStatus } from "../domain-status";

const base = { id: "d1", app_id: "a1", domain: "app.example.com", tls_ready: false, last_checked_at: null, created_at: "2026-10-16T00:00:00Z" };

describe("DomainStatus", () => {
  it("shows the pending message and a Check now button", () => {
    const onCheck = vi.fn();
    render(<DomainStatus domain={{ ...base, status: "pending", status_message: "DNS: no A/AAAA record for app.example.com; create an A record to 1.2.3.4" }} onCheck={onCheck} checking={false} />);
    expect(screen.getByText("pending")).toBeInTheDocument();
    expect(screen.getByText(/create an A record to 1\.2\.3\.4/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /check now/i }));
    expect(onCheck).toHaveBeenCalledTimes(1);
  });
  it("shows active with HTTPS and no message", () => {
    render(<DomainStatus domain={{ ...base, status: "active", tls_ready: true, status_message: "" }} onCheck={() => {}} checking={false} />);
    expect(screen.getByText("active")).toBeInTheDocument();
    expect(screen.getByText(/HTTPS ready/)).toBeInTheDocument();
  });
  it("disables the button while checking", () => {
    render(<DomainStatus domain={{ ...base, status: "verified", status_message: "DNS ok; waiting for the certificate" }} onCheck={() => {}} checking={true} />);
    expect(screen.getByRole("button", { name: /checking/i })).toBeDisabled();
  });
});

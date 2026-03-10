import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

// Mock framer-motion to avoid animation issues in tests
vi.mock("framer-motion", () => ({
  motion: new Proxy(
    {},
    {
      get: (_target, prop) => {
        // Return a forwardRef component for any HTML element (motion.div, motion.span, etc.)
        const Component = ({ children, ...props }: Record<string, unknown>) => {
          const { initial, animate, exit, variants, transition, whileInView, viewport, whileHover, whileTap, ...rest } = props;
          void initial; void animate; void exit; void variants; void transition; void whileInView; void viewport; void whileHover; void whileTap;
          const Tag = prop as string;
          return <Tag {...rest}>{children as React.ReactNode}</Tag>;
        };
        Component.displayName = `motion.${String(prop)}`;
        return Component;
      },
    }
  ),
  useInView: () => true,
  AnimatePresence: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

// Mock next/link
vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: { children: React.ReactNode; href: string }) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

// Mock all landing page components to isolate the page test
vi.mock("@/components/section", () => ({
  Section: ({ children, ...props }: Record<string, unknown>) => <section {...props}>{children as React.ReactNode}</section>,
  SectionHeader: ({ title, description }: { title: string; description?: string }) => (
    <div>
      <h2>{title}</h2>
      {description && <p>{description}</p>}
    </div>
  ),
}));

vi.mock("@/components/feature-card", () => ({
  FeatureCard: ({ title }: { title: string }) => <div data-testid="feature-card">{title}</div>,
}));

vi.mock("@/components/animated-terminal", () => ({
  AnimatedTerminal: () => <div data-testid="animated-terminal" />,
}));

vi.mock("@/components/trust-bar", () => ({
  TrustBar: () => <div data-testid="trust-bar" />,
}));

vi.mock("@/components/deploy-options", () => ({
  DeployOptions: () => <div data-testid="deploy-options" />,
}));

vi.mock("@/components/how-it-works", () => ({
  HowItWorks: () => <div data-testid="how-it-works" />,
}));

vi.mock("@/components/pricing-tabs", () => ({
  PricingTabs: () => <div data-testid="pricing-tabs" />,
}));

vi.mock("@/components/architecture-diagram", () => ({
  ArchitectureDiagram: () => <div data-testid="architecture-diagram" />,
}));

vi.mock("@/lib/urls", () => ({
  registerUrl: "https://app.freezenith.com/login?mode=register",
}));

import LandingPage from "../page";

describe("LandingPage", () => {
  it("renders the landing page", () => {
    render(<LandingPage />);
    // Hero section should contain the main tagline
    expect(screen.getByText(/ship faster/i) || screen.getByText(/scale/i)).toBeTruthy();
  });

  it("renders the pricing section", () => {
    render(<LandingPage />);
    expect(screen.getByTestId("pricing-tabs")).toBeInTheDocument();
  });

  it("renders feature cards", () => {
    render(<LandingPage />);
    const featureCards = screen.getAllByTestId("feature-card");
    expect(featureCards.length).toBeGreaterThan(0);
  });

  it("renders trust bar", () => {
    render(<LandingPage />);
    expect(screen.getByTestId("trust-bar")).toBeInTheDocument();
  });
});

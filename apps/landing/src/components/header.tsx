"use client";

import { useState } from "react";
import Link from "next/link";
import { cn } from "@/lib/utils";
import { Menu, X, Zap } from "lucide-react";

const navLinks = [
  { href: "#features", label: "Features" },
  { href: "#how-it-works", label: "How it Works" },
  { href: "#pricing", label: "Pricing" },
  { href: "/docs", label: "Docs" },
  { href: "https://github.com/DoTech/zenith", label: "GitHub" },
];

export function Header() {
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <header className="fixed top-0 z-50 w-full border-b border-border/50 bg-surface/80 backdrop-blur-xl">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3 sm:px-6">
        {/* Logo */}
        <Link href="/" className="flex items-center gap-2 group">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent-500/10 border border-accent-500/20 group-hover:bg-accent-500/20 transition-colors">
            <Zap className="h-4 w-4 text-accent-400" />
          </div>
          <span className="text-lg font-bold text-white">Zenith</span>
        </Link>

        {/* Desktop nav */}
        <nav className="hidden items-center gap-1 md:flex">
          {navLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="rounded-lg px-3 py-2 text-sm text-neutral-400 transition-colors hover:text-white hover:bg-surface-200"
              {...(link.href.startsWith("http") ? { target: "_blank", rel: "noopener noreferrer" } : {})}
            >
              {link.label}
            </Link>
          ))}
        </nav>

        {/* Desktop CTA */}
        <div className="hidden items-center gap-3 md:flex">
          <Link
            href="https://github.com/DoTech/zenith"
            className="rounded-lg px-4 py-2 text-sm text-neutral-300 transition-colors hover:text-white"
            target="_blank"
            rel="noopener noreferrer"
          >
            Star on GitHub
          </Link>
          <Link
            href="#get-started"
            className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white transition-all hover:bg-accent-600 hover:shadow-lg hover:shadow-accent-500/20"
          >
            Get Started
          </Link>
        </div>

        {/* Mobile toggle */}
        <button
          className="rounded-lg p-2 text-neutral-400 hover:text-white md:hidden"
          onClick={() => setMobileOpen(!mobileOpen)}
          aria-label="Toggle navigation"
        >
          {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </button>
      </div>

      {/* Mobile nav */}
      <div
        className={cn(
          "overflow-hidden border-t border-border/50 bg-surface/95 backdrop-blur-xl transition-all duration-300 md:hidden",
          mobileOpen ? "max-h-96 py-4" : "max-h-0 py-0"
        )}
      >
        <nav className="flex flex-col gap-1 px-4">
          {navLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              onClick={() => setMobileOpen(false)}
              className="rounded-lg px-3 py-2.5 text-sm text-neutral-400 transition-colors hover:text-white hover:bg-surface-200"
              {...(link.href.startsWith("http") ? { target: "_blank", rel: "noopener noreferrer" } : {})}
            >
              {link.label}
            </Link>
          ))}
          <div className="mt-2 border-t border-border pt-3">
            <Link
              href="#get-started"
              onClick={() => setMobileOpen(false)}
              className="block rounded-lg bg-accent-500 px-4 py-2.5 text-center text-sm font-medium text-white transition-all hover:bg-accent-600"
            >
              Get Started
            </Link>
          </div>
        </nav>
      </div>
    </header>
  );
}

import Link from "next/link";
import { Github, MessageCircle, Twitter } from "lucide-react";

const footerLinks = {
  Product: [
    { label: "Features", href: "/#features" },
    { label: "Pricing", href: "/pricing" },
    { label: "How it Works", href: "/#how-it-works" },
    { label: "Architecture", href: "/#architecture" },
    { label: "Documentation", href: "/docs" },
  ],
  Developers: [
    { label: "Getting Started", href: "/docs" },
    { label: "CLI Reference", href: "/docs" },
    { label: "API Docs", href: "/docs" },
    { label: "Helm Charts", href: "/docs" },
    { label: "GitHub", href: "https://github.com/DoTech/zenith" },
  ],
  Account: [
    { label: "Login", href: "https://app.freezenith.com/login" },
    { label: "Sign Up", href: "https://app.freezenith.com/register" },
    { label: "Dashboard", href: "https://app.freezenith.com" },
  ],
  Community: [
    { label: "Discord", href: "https://discord.gg/zenith" },
    { label: "Twitter", href: "https://twitter.com/freezenith" },
    { label: "Discussions", href: "https://github.com/DoTech/zenith/discussions" },
    { label: "Contributing", href: "https://github.com/DoTech/zenith/blob/main/CONTRIBUTING.md" },
    { label: "Changelog", href: "https://github.com/DoTech/zenith/releases" },
  ],
};

export function Footer() {
  return (
    <footer className="border-t border-border bg-surface">
      <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6 md:py-20">
        <div className="grid grid-cols-2 gap-8 md:grid-cols-6">
          {/* Brand column */}
          <div className="col-span-2">
            <Link href="/" className="flex items-center gap-2.5 group">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent-500 transition-all duration-300 group-hover:shadow-lg group-hover:shadow-accent-500/25">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                  <path d="M8 1L14 5V11L8 15L2 11V5L8 1Z" fill="white" fillOpacity="0.9" />
                  <path d="M8 1L14 5L8 9L2 5L8 1Z" fill="white" />
                </svg>
              </div>
              <span className="text-lg font-bold tracking-tight text-white">Zenith</span>
            </Link>
            <p className="mt-4 max-w-xs text-sm text-neutral-500 leading-relaxed">
              Cloud platform for developers. Deploy on Zenith Cloud or self-host the open-source PaaS.
            </p>
            <div className="mt-6 flex items-center gap-2">
              <Link
                href="https://github.com/DoTech/zenith"
                className="flex h-9 w-9 items-center justify-center rounded-lg text-neutral-500 transition-all hover:text-white hover:bg-surface-200"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="GitHub"
              >
                <Github className="h-4 w-4" />
              </Link>
              <Link
                href="https://discord.gg/zenith"
                className="flex h-9 w-9 items-center justify-center rounded-lg text-neutral-500 transition-all hover:text-white hover:bg-surface-200"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Discord"
              >
                <MessageCircle className="h-4 w-4" />
              </Link>
              <Link
                href="https://twitter.com/freezenith"
                className="flex h-9 w-9 items-center justify-center rounded-lg text-neutral-500 transition-all hover:text-white hover:bg-surface-200"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Twitter"
              >
                <Twitter className="h-4 w-4" />
              </Link>
            </div>
          </div>

          {/* Link columns */}
          {Object.entries(footerLinks).map(([category, links]) => (
            <div key={category}>
              <h3 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
                {category}
              </h3>
              <ul className="mt-4 space-y-2.5">
                {links.map((link) => (
                  <li key={link.label}>
                    <Link
                      href={link.href}
                      className="text-sm text-neutral-500 transition-colors hover:text-neutral-200"
                      {...(link.href.startsWith("http")
                        ? { target: "_blank", rel: "noopener noreferrer" }
                        : {})}
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom bar */}
        <div className="mt-16 flex flex-col items-center justify-between gap-4 border-t border-border pt-8 md:flex-row">
          <p className="text-xs text-neutral-600">
            Made by{" "}
            <Link
              href="https://dotech.com"
              className="text-neutral-500 hover:text-white transition-colors"
              target="_blank"
              rel="noopener noreferrer"
            >
              DoTech
            </Link>
            . MIT Licensed. Open source on GitHub.
          </p>
          <p className="text-xs text-neutral-600">freezenith.com</p>
        </div>
      </div>
    </footer>
  );
}

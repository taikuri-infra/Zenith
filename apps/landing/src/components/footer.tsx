import Link from "next/link";
import { Zap, Github, MessageCircle } from "lucide-react";

const footerLinks = {
  Product: [
    { label: "Features", href: "#features" },
    { label: "Pricing", href: "#pricing" },
    { label: "How it Works", href: "#how-it-works" },
    { label: "Architecture", href: "#architecture" },
  ],
  Resources: [
    { label: "Documentation", href: "/docs" },
    { label: "GitHub", href: "https://github.com/DoTech/zenith" },
    { label: "Changelog", href: "https://github.com/DoTech/zenith/releases" },
    { label: "Contributing", href: "https://github.com/DoTech/zenith/blob/main/CONTRIBUTING.md" },
  ],
  Community: [
    { label: "Discord", href: "https://discord.gg/zenith" },
    { label: "Twitter", href: "https://twitter.com/freezenith" },
    { label: "GitHub Discussions", href: "https://github.com/DoTech/zenith/discussions" },
  ],
};

export function Footer() {
  return (
    <footer className="border-t border-border bg-surface">
      <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6 md:py-16">
        <div className="grid grid-cols-2 gap-8 md:grid-cols-4">
          {/* Brand column */}
          <div className="col-span-2 md:col-span-1">
            <Link href="/" className="flex items-center gap-2 group">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent-500/10 border border-accent-500/20">
                <Zap className="h-4 w-4 text-accent-400" />
              </div>
              <span className="text-lg font-bold text-white">Zenith</span>
            </Link>
            <p className="mt-4 text-sm text-neutral-500 max-w-xs">
              100% free, open-source Kubernetes PaaS. Deploy everything with a single command.
            </p>
            <div className="mt-4 flex items-center gap-3">
              <Link
                href="https://github.com/DoTech/zenith"
                className="rounded-lg p-2 text-neutral-500 transition-colors hover:text-white hover:bg-surface-200"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="GitHub"
              >
                <Github className="h-4 w-4" />
              </Link>
              <Link
                href="https://discord.gg/zenith"
                className="rounded-lg p-2 text-neutral-500 transition-colors hover:text-white hover:bg-surface-200"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Discord"
              >
                <MessageCircle className="h-4 w-4" />
              </Link>
            </div>
          </div>

          {/* Link columns */}
          {Object.entries(footerLinks).map(([category, links]) => (
            <div key={category}>
              <h3 className="text-sm font-semibold text-white">{category}</h3>
              <ul className="mt-4 space-y-2.5">
                {links.map((link) => (
                  <li key={link.label}>
                    <Link
                      href={link.href}
                      className="text-sm text-neutral-500 transition-colors hover:text-neutral-300"
                      {...(link.href.startsWith("http") ? { target: "_blank", rel: "noopener noreferrer" } : {})}
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
        <div className="mt-12 flex flex-col items-center justify-between gap-4 border-t border-border pt-8 md:flex-row">
          <p className="text-xs text-neutral-600">
            Made by{" "}
            <Link href="https://dotech.com" className="text-neutral-500 hover:text-white transition-colors" target="_blank" rel="noopener noreferrer">
              DoTech
            </Link>
            {" "}. MIT Licensed.
          </p>
          <p className="text-xs text-neutral-600">
            freezenith.com
          </p>
        </div>
      </div>
    </footer>
  );
}

import Link from "next/link";
import { Github } from "lucide-react";
import { site } from "@/lib/site";
import { Logo } from "./logo";

const columns = {
  Project: [
    { label: "What it is", href: "#what" },
    { label: "The stack", href: "#stack" },
    { label: "Features", href: "#features" },
    { label: "Bring your own infra", href: "#infra" },
  ],
  "Get started": [
    { label: "Self-host guide", href: "#quickstart" },
    { label: "Source code", href: site.githubUrl },
    { label: "Issues", href: `${site.githubUrl}/issues` },
    { label: "Discussions", href: `${site.githubUrl}/discussions` },
  ],
};

export function Footer() {
  return (
    <footer className="border-t border-border bg-surface">
      <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6 md:py-20">
        <div className="grid grid-cols-2 gap-8 md:grid-cols-4">
          <div className="col-span-2">
            <Logo />
            <p className="mt-4 max-w-xs text-sm leading-relaxed text-neutral-500">
              A source-available internal developer platform, free to self-host. Run a full
              private cloud on your own infrastructure — no vendor, no lock-in, no bill.
            </p>
            <Link
              href={site.githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              aria-label="GitHub"
              className="mt-6 inline-flex h-9 w-9 items-center justify-center rounded-lg text-neutral-500 transition-all hover:bg-surface-200 hover:text-white"
            >
              <Github className="h-4 w-4" />
            </Link>
          </div>

          {Object.entries(columns).map(([title, links]) => (
            <div key={title}>
              <h3 className="font-mono text-xs font-semibold uppercase tracking-wider text-neutral-400">
                {title}
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

        <div className="mt-16 flex flex-col items-center justify-between gap-4 border-t border-border pt-8 md:flex-row">
          <p className="text-xs text-neutral-600">
            {site.license} licensed · Free to self-host · Source available.
          </p>
          <p className="font-mono text-xs text-neutral-600">freezenith.com</p>
        </div>
      </div>
    </footer>
  );
}

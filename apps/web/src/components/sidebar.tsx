"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Boxes,
  Database,
  HardDrive,
  Network,
  Globe,
  Shield,
  KeyRound,
  Activity,
  Container,
  Server,
  BookOpen,
  CreditCard,
  Settings,
  ScrollText,
  History,
  ListOrdered,
} from "lucide-react";

const isStandalone = process.env.NEXT_PUBLIC_ZENITH_MODE !== "saas";

interface NavItem {
  name: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  saasOnly?: boolean;
}

interface NavSection {
  label: string;
  items: NavItem[];
  saasOnly?: boolean;
}

const navSections: NavSection[] = [
  {
    label: "OVERVIEW",
    items: [
      { name: "Overview", href: "/", icon: LayoutDashboard },
    ],
  },
  {
    label: "COMPUTE",
    items: [
      { name: "Apps", href: "/apps", icon: Boxes },
      { name: "Databases", href: "/databases", icon: Database },
      { name: "Storage", href: "/storage", icon: HardDrive },
      { name: "Queues", href: "/queues", icon: ListOrdered },
    ],
  },
  {
    label: "NETWORKING",
    items: [
      { name: "Gateway", href: "/gateway", icon: Network, saasOnly: true },
      { name: "Domains", href: "/networking", icon: Globe },
    ],
  },
  {
    label: "SECURITY",
    items: [
      { name: "Auth", href: "/auth", icon: Shield },
      { name: "IAM", href: "/iam", icon: KeyRound },
    ],
  },
  {
    label: "OBSERVABILITY",
    items: [
      { name: "Logs", href: "/logs", icon: ScrollText },
      { name: "Activity", href: "/activity", icon: History },
      { name: "Monitoring", href: "/monitoring", icon: Activity },
      { name: "Registry", href: "/registry", icon: Container },
    ],
  },
  {
    label: "INFRASTRUCTURE",
    saasOnly: true,
    items: [
      { name: "Planets", href: "/planets", icon: Server, saasOnly: true },
    ],
  },
];

const bottomNav: NavItem[] = [
  { name: "Docs", href: "/docs", icon: BookOpen },
  { name: "Billing", href: "/billing", icon: CreditCard, saasOnly: true },
  { name: "Settings", href: "/settings", icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();

  const isActive = (href: string) =>
    href === "/" ? pathname === "/" : pathname.startsWith(href);

  return (
    <aside className="fixed left-0 top-0 z-40 flex h-screen w-56 flex-col border-r border-border bg-surface-50">
      {/* Logo */}
      <div className="flex h-14 items-center gap-2.5 border-b border-border px-4">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-accent-500 text-xs font-bold text-white">
          Z
        </div>
        <span className="text-sm font-semibold text-white">Zenith</span>
      </div>

      {/* Project selector */}
      <div className="border-b border-border px-3 py-2.5">
        <button className="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-sm text-neutral-300 transition-colors hover:bg-surface-300">
          <span className="truncate font-medium">my-startup</span>
          <svg className="h-4 w-4 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l4-4 4 4m0 6l-4 4-4-4" />
          </svg>
        </button>
      </div>

      {/* Main nav with sections */}
      <nav className="flex-1 overflow-y-auto px-3 py-3">
        {navSections
          .filter((s) => !isStandalone || !s.saasOnly)
          .map((section, sectionIdx) => {
            const items = section.items.filter(
              (item) => !isStandalone || !item.saasOnly
            );
            if (items.length === 0) return null;
            return (
              <div key={section.label} className={sectionIdx > 0 ? "mt-4" : ""}>
                <div className="mb-1 px-2.5 text-[11px] font-semibold uppercase tracking-wider text-neutral-600">
                  {section.label}
                </div>
                <div className="space-y-0.5">
                  {items.map((item) => (
                    <Link
                      key={item.name}
                      href={item.href}
                      className={`flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors ${
                        isActive(item.href)
                          ? "bg-accent-500/15 text-accent-400 font-medium"
                          : "text-neutral-400 hover:bg-surface-300 hover:text-white"
                      }`}
                    >
                      <item.icon className="h-4 w-4 flex-shrink-0" />
                      {item.name}
                    </Link>
                  ))}
                </div>
              </div>
            );
          })}
      </nav>

      {/* Bottom nav */}
      <div className="border-t border-border px-3 py-3 space-y-0.5">
        {bottomNav
          .filter((item) => !isStandalone || !item.saasOnly)
          .map((item) => (
            <Link
              key={item.name}
              href={item.href}
              className={`flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors ${
                isActive(item.href)
                  ? "bg-accent-500/15 text-accent-400 font-medium"
                  : "text-neutral-400 hover:bg-surface-300 hover:text-white"
              }`}
            >
              <item.icon className="h-4 w-4 flex-shrink-0" />
              {item.name}
            </Link>
          ))}
      </div>
    </aside>
  );
}

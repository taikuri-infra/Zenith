"use client";

import { useState, useRef, useEffect } from "react";
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
  LifeBuoy,
  ScrollText,
  History,
  ListOrdered,
  Plus,
  Check,
} from "lucide-react";
import { useProjectContext } from "@/hooks/use-project";

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
  { name: "Support", href: "/support", icon: LifeBuoy, saasOnly: true },
  { name: "Docs", href: "/docs", icon: BookOpen },
  { name: "Billing", href: "/billing", icon: CreditCard, saasOnly: true },
  { name: "Settings", href: "/settings", icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();
  const { currentProject, projects, setCurrentProject, createProject } = useProjectContext();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false);
        setCreating(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const handleCreate = async () => {
    if (!newName.trim()) return;
    try {
      const p = await createProject(newName.trim());
      setCurrentProject(p);
      setNewName("");
      setCreating(false);
      setDropdownOpen(false);
    } catch {
      // ignore
    }
  };

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
      <div className="relative border-b border-border px-3 py-2.5" ref={dropdownRef}>
        <button
          onClick={() => setDropdownOpen(!dropdownOpen)}
          className="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-sm text-neutral-300 transition-colors hover:bg-surface-300"
        >
          <span className="truncate font-medium">{currentProject?.name || "Select project"}</span>
          <svg className="h-4 w-4 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l4-4 4 4m0 6l-4 4-4-4" />
          </svg>
        </button>

        {dropdownOpen && (
          <div className="absolute left-2 right-2 top-full z-50 mt-1 rounded-md border border-border bg-surface-100 py-1 shadow-lg">
            {projects.map((p) => (
              <button
                key={p.id}
                onClick={() => {
                  setCurrentProject(p);
                  setDropdownOpen(false);
                }}
                className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-neutral-300 hover:bg-surface-300"
              >
                {p.id === currentProject?.id && <Check className="h-3 w-3 text-accent-400" />}
                {p.id !== currentProject?.id && <span className="w-3" />}
                <span className="truncate">{p.name}</span>
              </button>
            ))}
            <div className="border-t border-border mt-1 pt-1">
              {creating ? (
                <div className="px-3 py-1.5">
                  <input
                    autoFocus
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                    placeholder="Project name"
                    className="w-full rounded bg-surface-300 px-2 py-1 text-sm text-white placeholder-neutral-500 outline-none focus:ring-1 focus:ring-accent-500"
                  />
                </div>
              ) : (
                <button
                  onClick={() => setCreating(true)}
                  className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-neutral-400 hover:bg-surface-300 hover:text-white"
                >
                  <Plus className="h-3 w-3" />
                  Create Project
                </button>
              )}
            </div>
          </div>
        )}
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

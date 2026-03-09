"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import {
  LayoutDashboard,
  BarChart3,
  Building2,
  CreditCard,
  Target,
  LifeBuoy,
  Award,
  Server,
  HardDrive,
  Database,
  HardDriveDownload,
  Network,
  LineChart,
  ScrollText,
  Bell,
  Activity,
  Shield,
  ShieldAlert,
  ScanLine,
  Users2,
  Package,
  Archive,
  GitBranch,
  Container,
  Settings,
  ChevronDown,
  Users,
} from "lucide-react";

interface NavItem {
  name: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
}

interface NavGroup {
  label: string;
  items: NavItem[];
}

const navGroups: NavGroup[] = [
  {
    label: "WAR ROOM",
    items: [
      { name: "Command Center", href: "/", icon: LayoutDashboard },
    ],
  },
  {
    label: "BUSINESS",
    items: [
      { name: "Analytics", href: "/analytics", icon: BarChart3 },
      { name: "Customers", href: "/customers", icon: Building2 },
      { name: "Plans & Pricing", href: "/plans", icon: CreditCard },
      { name: "CRM Pipeline", href: "/crm", icon: Target },
    ],
  },
  {
    label: "SUPPORT",
    items: [
      { name: "Tickets", href: "/support", icon: LifeBuoy },
      { name: "Quality & SLA", href: "/quality", icon: Award },
    ],
  },
  {
    label: "INFRASTRUCTURE",
    items: [
      { name: "Services", href: "/services", icon: Server },
      { name: "Clusters", href: "/clusters", icon: HardDrive },
      { name: "Nodes & Compute", href: "/infrastructure", icon: HardDriveDownload },
      { name: "Databases", href: "/databases", icon: Database },
      { name: "Storage", href: "/storage", icon: Archive },
      { name: "Networking", href: "/networking", icon: Network },
    ],
  },
  {
    label: "OBSERVABILITY",
    items: [
      { name: "Dashboards", href: "/dashboards", icon: LineChart },
      { name: "Logs", href: "/logs", icon: ScrollText },
      { name: "Alerts", href: "/alerts", icon: Bell },
      { name: "Traces", href: "/traces", icon: Activity },
    ],
  },
  {
    label: "SECURITY",
    items: [
      { name: "Overview", href: "/security", icon: Shield },
      { name: "WAF & Policies", href: "/security/waf", icon: ShieldAlert },
      { name: "Image Scanning", href: "/security/images", icon: ScanLine },
      { name: "Audit Log", href: "/audit", icon: ScrollText },
      { name: "Sessions", href: "/security/sessions", icon: Users2 },
    ],
  },
  {
    label: "PLATFORM",
    items: [
      { name: "Modules", href: "/modules", icon: Package },
      { name: "Backups", href: "/backups", icon: HardDriveDownload },
      { name: "GitOps", href: "/gitops", icon: GitBranch },
      { name: "Registry", href: "/registry", icon: Container },
      { name: "Admin Users", href: "/admin-users", icon: Users },
    ],
  },
];

const bottomNav: NavItem[] = [
  { name: "Settings", href: "/settings", icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  const toggleGroup = (label: string) => {
    setCollapsed((prev) => ({ ...prev, [label]: !prev[label] }));
  };

  const isActive = (href: string) => {
    if (href === "/") return pathname === "/";
    return pathname === href || pathname.startsWith(href + "/");
  };

  return (
    <aside className="fixed left-0 top-0 z-40 flex h-screen w-60 flex-col border-r border-border bg-surface-50">
      {/* Logo */}
      <div className="flex h-14 items-center gap-2.5 border-b border-border px-5">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-accent-600 text-xs font-bold text-white">
          Z
        </div>
        <div>
          <span className="text-sm font-semibold text-white">Mission Control</span>
        </div>
      </div>

      {/* Main nav */}
      <nav className="flex-1 overflow-y-auto px-3 py-2">
        {navGroups.map((group) => {
          const isCollapsed = collapsed[group.label];
          return (
            <div key={group.label} className="mb-1">
              <button
                onClick={() => toggleGroup(group.label)}
                className="flex w-full items-center justify-between px-2 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-neutral-500 hover:text-neutral-400"
              >
                {group.label}
                <ChevronDown
                  className={`h-3 w-3 transition-transform ${
                    isCollapsed ? "-rotate-90" : ""
                  }`}
                />
              </button>
              {!isCollapsed && (
                <div className="space-y-0.5">
                  {group.items.map((item) => (
                    <Link
                      key={item.href}
                      href={item.href}
                      className={`flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition-colors ${
                        isActive(item.href)
                          ? "bg-accent-600/15 text-accent-400 font-medium"
                          : "text-neutral-400 hover:bg-surface-300 hover:text-white"
                      }`}
                    >
                      <item.icon className="h-4 w-4 flex-shrink-0" />
                      {item.name}
                    </Link>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </nav>

      {/* Bottom nav */}
      <div className="border-t border-border px-3 py-3">
        {bottomNav.map((item) => (
          <Link
            key={item.name}
            href={item.href}
            className={`flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors ${
              isActive(item.href)
                ? "bg-accent-600/15 text-accent-400 font-medium"
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

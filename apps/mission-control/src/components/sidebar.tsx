"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Server,
  Building2,
  CreditCard,
  Package,
  ArrowUpCircle,
  Users,
  HardDrive,
  Database,
  ScrollText,
  Settings,
  LifeBuoy,
} from "lucide-react";

const navigation = [
  { name: "Dashboard", href: "/", icon: LayoutDashboard },
  { name: "Customers", href: "/customers", icon: Building2 },
  { name: "Support", href: "/support", icon: LifeBuoy },
  { name: "Plans", href: "/plans", icon: CreditCard },
  { name: "Clusters", href: "/clusters", icon: Server },
  { name: "Modules", href: "/modules", icon: Package },
  { name: "Updates", href: "/updates", icon: ArrowUpCircle },
  { name: "Tenants", href: "/tenants", icon: Users },
  { name: "Infrastructure", href: "/infrastructure", icon: HardDrive },
  { name: "State", href: "/state", icon: Database },
  { name: "Audit Log", href: "/audit", icon: ScrollText },
];

const bottomNav = [
  { name: "Settings", href: "/settings", icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();

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
      <nav className="flex-1 space-y-0.5 px-3 py-3">
        {navigation.map((item) => {
          const isActive =
            item.href === "/"
              ? pathname === "/"
              : pathname.startsWith(item.href);
          return (
            <Link
              key={item.name}
              href={item.href}
              className={`flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors ${
                isActive
                  ? "bg-accent-600/15 text-accent-400 font-medium"
                  : "text-neutral-400 hover:bg-surface-300 hover:text-white"
              }`}
            >
              <item.icon className="h-4 w-4 flex-shrink-0" />
              {item.name}
            </Link>
          );
        })}
      </nav>

      {/* Bottom nav */}
      <div className="border-t border-border px-3 py-3">
        {bottomNav.map((item) => {
          const isActive = pathname.startsWith(item.href);
          return (
            <Link
              key={item.name}
              href={item.href}
              className={`flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors ${
                isActive
                  ? "bg-accent-600/15 text-accent-400 font-medium"
                  : "text-neutral-400 hover:bg-surface-300 hover:text-white"
              }`}
            >
              <item.icon className="h-4 w-4 flex-shrink-0" />
              {item.name}
            </Link>
          );
        })}
      </div>
    </aside>
  );
}

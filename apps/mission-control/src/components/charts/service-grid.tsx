"use client";

interface ServiceItem {
  name: string;
  status: "healthy" | "degraded" | "down" | "unknown";
}

interface ServiceGridProps {
  services: ServiceItem[];
}

const statusColors: Record<string, string> = {
  healthy: "bg-emerald-400",
  degraded: "bg-amber-400",
  down: "bg-red-400",
  unknown: "bg-neutral-500",
};

export function ServiceGrid({ services }: ServiceGridProps) {
  return (
    <div className="grid grid-cols-4 gap-2">
      {services.map((svc) => (
        <div
          key={svc.name}
          className="flex items-center gap-2 rounded-md border border-border bg-surface-100 px-2.5 py-1.5"
        >
          <div
            className={`h-2 w-2 rounded-full ${
              statusColors[svc.status] || statusColors.unknown
            }`}
          />
          <span className="truncate text-xs text-neutral-300">{svc.name}</span>
        </div>
      ))}
    </div>
  );
}

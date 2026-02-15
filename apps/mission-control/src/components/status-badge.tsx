const statusStyles = {
  healthy: "bg-emerald-500/10 text-emerald-400",
  warning: "bg-amber-500/10 text-amber-400",
  error: "bg-red-500/10 text-red-400",
  active: "bg-emerald-500/10 text-emerald-400",
  idle: "bg-neutral-500/10 text-neutral-400",
  suspended: "bg-red-500/10 text-red-400",
  up_to_date: "bg-emerald-500/10 text-emerald-400",
  update_available: "bg-accent-500/10 text-accent-400",
} as const;

const statusDots = {
  healthy: "bg-emerald-400",
  warning: "bg-amber-400",
  error: "bg-red-400",
  active: "bg-emerald-400",
  idle: "bg-neutral-400",
  suspended: "bg-red-400",
  up_to_date: "bg-emerald-400",
  update_available: "bg-accent-400",
} as const;

type Status = keyof typeof statusStyles;

const labels: Record<Status, string> = {
  healthy: "Healthy",
  warning: "Warning",
  error: "Error",
  active: "Active",
  idle: "Idle",
  suspended: "Suspended",
  up_to_date: "Up to date",
  update_available: "Update",
};

interface StatusBadgeProps {
  status: Status;
  label?: string;
}

export function StatusBadge({ status, label }: StatusBadgeProps) {
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${statusStyles[status]}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${statusDots[status]}`} />
      {label || labels[status]}
    </span>
  );
}

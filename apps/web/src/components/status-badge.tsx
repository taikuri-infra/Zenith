const styles = {
  running: "bg-emerald-500/10 text-emerald-400",
  deploying: "bg-amber-500/10 text-amber-400",
  stopped: "bg-neutral-500/10 text-neutral-400",
  crashed: "bg-red-500/10 text-red-400",
  creating: "bg-amber-500/10 text-amber-400",
  active: "bg-emerald-500/10 text-emerald-400",
  pending: "bg-amber-500/10 text-amber-400",
  error: "bg-red-500/10 text-red-400",
  ready: "bg-emerald-500/10 text-emerald-400",
  joining: "bg-amber-500/10 text-amber-400",
  draining: "bg-amber-500/10 text-amber-400",
  live: "bg-emerald-500/10 text-emerald-400",
  building: "bg-accent-500/10 text-accent-400",
  failed: "bg-red-500/10 text-red-400",
  superseded: "bg-neutral-500/10 text-neutral-400",
  sleeping: "bg-indigo-500/10 text-indigo-400",
  provisioning: "bg-amber-500/10 text-amber-400",
  deleting: "bg-neutral-500/10 text-neutral-400",
} as const;

const dots = {
  running: "bg-emerald-400",
  deploying: "bg-amber-400",
  stopped: "bg-neutral-400",
  crashed: "bg-red-400",
  creating: "bg-amber-400",
  active: "bg-emerald-400",
  pending: "bg-amber-400",
  error: "bg-red-400",
  ready: "bg-emerald-400",
  joining: "bg-amber-400",
  draining: "bg-amber-400",
  live: "bg-emerald-400",
  building: "bg-accent-400",
  failed: "bg-red-400",
  superseded: "bg-neutral-400",
  sleeping: "bg-indigo-400 animate-pulse",
  provisioning: "bg-amber-400 animate-pulse",
  deleting: "bg-neutral-400 animate-pulse",
} as const;

type Status = keyof typeof styles;

export function StatusBadge({ status }: { status: Status }) {
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium capitalize ${styles[status]}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${dots[status]}`} />
      {status}
    </span>
  );
}

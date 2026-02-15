interface ProgressBarProps {
  percent: number;
  label?: string;
  size?: "sm" | "md";
}

export function ProgressBar({ percent, label, size = "sm" }: ProgressBarProps) {
  const colorClass =
    percent >= 80
      ? "bg-red-500"
      : percent >= 60
        ? "bg-amber-500"
        : "bg-accent-500";

  return (
    <div className="flex items-center gap-2">
      <div className={`flex-1 overflow-hidden rounded-full bg-surface-400 ${size === "sm" ? "h-1.5" : "h-2.5"}`}>
        <div
          className={`h-full rounded-full transition-all ${colorClass}`}
          style={{ width: `${Math.min(percent, 100)}%` }}
        />
      </div>
      {label && <span className="text-xs tabular-nums text-neutral-400">{label}</span>}
    </div>
  );
}

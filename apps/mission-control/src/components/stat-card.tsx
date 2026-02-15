interface StatCardProps {
  label: string;
  value: string | number;
  sub?: string;
  alert?: boolean;
}

export function StatCard({ label, value, sub, alert }: StatCardProps) {
  return (
    <div className="rounded-lg border border-border bg-surface-100 p-4">
      <p className="text-xs font-medium text-neutral-500">{label}</p>
      <p className="mt-1 text-2xl font-semibold text-white">{value}</p>
      {sub && (
        <p className={`mt-0.5 text-xs ${alert ? "text-amber-400" : "text-neutral-500"}`}>
          {alert && "⚠ "}{sub}
        </p>
      )}
    </div>
  );
}

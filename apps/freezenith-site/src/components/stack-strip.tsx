const items = [
  "k3s / Kubernetes",
  "Backstage",
  "Cilium",
  "Kyverno",
  "Velero",
  "VictoriaMetrics",
  "GitOps",
  "PostgreSQL",
  "Object storage",
];

export function StackStrip() {
  return (
    <div className="relative border-y border-border/60 bg-surface-50/40 py-6">
      <div className="pointer-events-none absolute inset-y-0 left-0 z-10 w-24 bg-gradient-to-r from-surface to-transparent" />
      <div className="pointer-events-none absolute inset-y-0 right-0 z-10 w-24 bg-gradient-to-l from-surface to-transparent" />
      <div className="flex overflow-hidden">
        <div className="flex shrink-0 animate-marquee items-center gap-10 pr-10">
          {[...items, ...items].map((item, i) => (
            <span
              key={i}
              className="whitespace-nowrap font-mono text-sm text-neutral-500"
            >
              {item}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}

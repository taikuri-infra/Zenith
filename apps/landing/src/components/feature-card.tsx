import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";

interface FeatureCardProps {
  icon: LucideIcon;
  title: string;
  description: string;
  className?: string;
}

export function FeatureCard({ icon: Icon, title, description, className }: FeatureCardProps) {
  return (
    <div
      className={cn(
        "group relative rounded-xl border border-border bg-surface-50 p-6 transition-all duration-300",
        "hover:border-accent-500/30 hover:bg-surface-100",
        "glow-border",
        className
      )}
    >
      <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-accent-500/10 border border-accent-500/20 transition-colors group-hover:bg-accent-500/15">
        <Icon className="h-5 w-5 text-accent-400" />
      </div>
      <h3 className="mb-2 text-base font-semibold text-white">{title}</h3>
      <p className="text-sm leading-relaxed text-neutral-400">{description}</p>
    </div>
  );
}

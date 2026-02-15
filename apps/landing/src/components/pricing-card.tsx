import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import Link from "next/link";

interface PricingCardProps {
  name: string;
  price: string;
  period?: string;
  description: string;
  features: string[];
  cta: string;
  ctaHref: string;
  featured?: boolean;
  className?: string;
}

export function PricingCard({
  name,
  price,
  period,
  description,
  features,
  cta,
  ctaHref,
  featured = false,
  className,
}: PricingCardProps) {
  return (
    <div
      className={cn(
        "relative rounded-xl border p-6 md:p-8 transition-all duration-300",
        featured
          ? "border-accent-500/40 bg-accent-500/5"
          : "border-border bg-surface-50 hover:border-border-hover",
        className
      )}
    >
      {featured && (
        <div className="absolute -top-3 left-1/2 -translate-x-1/2">
          <span className="rounded-full bg-accent-500 px-3 py-1 text-xs font-semibold text-white">
            Most Popular
          </span>
        </div>
      )}

      <div className="mb-6">
        <h3 className="text-lg font-semibold text-white">{name}</h3>
        <p className="mt-1 text-sm text-neutral-500">{description}</p>
      </div>

      <div className="mb-6">
        <span className="text-4xl font-bold text-white">{price}</span>
        {period && <span className="ml-1 text-sm text-neutral-500">{period}</span>}
      </div>

      <ul className="mb-8 space-y-3">
        {features.map((feature) => (
          <li key={feature} className="flex items-start gap-2.5">
            <Check className="mt-0.5 h-4 w-4 shrink-0 text-accent-400" />
            <span className="text-sm text-neutral-300">{feature}</span>
          </li>
        ))}
      </ul>

      <Link
        href={ctaHref}
        className={cn(
          "block w-full rounded-lg py-2.5 text-center text-sm font-medium transition-all",
          featured
            ? "bg-accent-500 text-white hover:bg-accent-600 hover:shadow-lg hover:shadow-accent-500/20"
            : "border border-border bg-surface-200 text-neutral-300 hover:border-border-hover hover:text-white"
        )}
      >
        {cta}
      </Link>
    </div>
  );
}

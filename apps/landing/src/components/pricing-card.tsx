"use client";

import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import Link from "next/link";
import { motion, useInView } from "framer-motion";
import { useRef } from "react";

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
  index?: number;
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
  index = 0,
}: PricingCardProps) {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-80px" });

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 30 }}
      animate={isInView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.5, delay: index * 0.1 }}
      className={cn(
        "relative rounded-2xl border p-7 md:p-8 transition-all duration-500",
        featured
          ? "border-accent-500/30 pricing-featured shadow-lg shadow-accent-500/5"
          : "border-border bg-surface-50/50 hover:border-border-hover",
        className
      )}
    >
      {featured && (
        <div className="absolute -top-3 left-1/2 -translate-x-1/2">
          <span className="rounded-full bg-accent-500 px-3.5 py-1 text-xs font-semibold text-white shadow-lg shadow-accent-500/25">
            Most Popular
          </span>
        </div>
      )}

      <div className="mb-6">
        <h3 className="text-lg font-semibold text-white">{name}</h3>
        <p className="mt-1.5 text-sm text-neutral-500 leading-relaxed">{description}</p>
      </div>

      <div className="mb-8">
        <span className="text-4xl font-bold tracking-tight text-white">{price}</span>
        {period && <span className="ml-1.5 text-sm text-neutral-500">{period}</span>}
      </div>

      <ul className="mb-8 space-y-3.5">
        {features.map((feature) => (
          <li key={feature} className="flex items-start gap-3">
            <div className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-accent-500/10">
              <Check className="h-3 w-3 text-accent-400" />
            </div>
            <span className="text-sm text-neutral-300 leading-relaxed">{feature}</span>
          </li>
        ))}
      </ul>

      <Link
        href={ctaHref}
        className={cn(
          "block w-full rounded-xl py-3 text-center text-sm font-medium transition-all duration-300",
          featured
            ? "bg-accent-500 text-white hover:bg-accent-600 hover:shadow-lg hover:shadow-accent-500/25"
            : "border border-border bg-surface-200 text-neutral-300 hover:border-border-hover hover:text-white hover:bg-surface-300"
        )}
      >
        {cta}
      </Link>
    </motion.div>
  );
}

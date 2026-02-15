"use client";

import { cn } from "@/lib/utils";
import { motion, useInView } from "framer-motion";
import { useRef } from "react";
import type { LucideIcon } from "lucide-react";

interface FeatureCardProps {
  icon: LucideIcon;
  title: string;
  description: string;
  className?: string;
  index?: number;
}

export function FeatureCard({ icon: Icon, title, description, className, index = 0 }: FeatureCardProps) {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-80px" });

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 30 }}
      animate={isInView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.5, delay: index * 0.08 }}
      className={cn(
        "group relative rounded-2xl border border-border bg-surface-50/50 p-6 md:p-7 transition-all duration-500",
        "hover:border-accent-500/20 hover:bg-surface-100/80",
        "hover:shadow-lg hover:shadow-accent-500/5",
        className
      )}
    >
      {/* Subtle gradient on hover */}
      <div className="absolute inset-0 rounded-2xl bg-gradient-to-br from-accent-500/[0.03] to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500" />

      <div className="relative">
        <div className="mb-5 flex h-11 w-11 items-center justify-center rounded-xl bg-accent-500/10 border border-accent-500/15 transition-all duration-300 group-hover:bg-accent-500/15 group-hover:border-accent-500/25 group-hover:shadow-lg group-hover:shadow-accent-500/10">
          <Icon className="h-5 w-5 text-accent-400 transition-transform duration-300 group-hover:scale-110" />
        </div>
        <h3 className="mb-2.5 text-base font-semibold text-white">{title}</h3>
        <p className="text-sm leading-relaxed text-neutral-400">{description}</p>
      </div>
    </motion.div>
  );
}

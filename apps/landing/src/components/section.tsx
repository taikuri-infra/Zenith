"use client";

import { cn } from "@/lib/utils";
import { motion } from "framer-motion";
import { useRef } from "react";
import { useInView } from "framer-motion";

interface SectionProps {
  children: React.ReactNode;
  className?: string;
  id?: string;
}

export function Section({ children, className, id }: SectionProps) {
  return (
    <section id={id} className={cn("relative py-24 md:py-32 px-4 sm:px-6", className)}>
      <div className="mx-auto max-w-6xl">{children}</div>
    </section>
  );
}

interface SectionHeaderProps {
  label?: string;
  title: string;
  description?: string;
  className?: string;
}

export function SectionHeader({ label, title, description, className }: SectionHeaderProps) {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-100px" });

  return (
    <div ref={ref} className={cn("mb-16 md:mb-20 text-center", className)}>
      {label && (
        <motion.span
          initial={{ opacity: 0, y: 10 }}
          animate={isInView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5 }}
          className="mb-4 inline-block rounded-full border border-accent-500/20 bg-accent-500/5 px-4 py-1.5 text-xs font-medium tracking-wide uppercase text-accent-400"
        >
          {label}
        </motion.span>
      )}
      <motion.h2
        initial={{ opacity: 0, y: 20 }}
        animate={isInView ? { opacity: 1, y: 0 } : {}}
        transition={{ duration: 0.5, delay: 0.1 }}
        className="text-3xl font-bold tracking-tight text-white md:text-4xl lg:text-5xl"
      >
        {title}
      </motion.h2>
      {description && (
        <motion.p
          initial={{ opacity: 0, y: 20 }}
          animate={isInView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="mx-auto mt-5 max-w-2xl text-base text-neutral-400 md:text-lg leading-relaxed"
        >
          {description}
        </motion.p>
      )}
    </div>
  );
}

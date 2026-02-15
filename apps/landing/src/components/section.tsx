import { cn } from "@/lib/utils";

interface SectionProps {
  children: React.ReactNode;
  className?: string;
  id?: string;
}

export function Section({ children, className, id }: SectionProps) {
  return (
    <section id={id} className={cn("py-20 md:py-28 px-4 sm:px-6", className)}>
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
  return (
    <div className={cn("mb-12 md:mb-16 text-center", className)}>
      {label && (
        <span className="mb-3 inline-block rounded-full border border-accent-500/30 bg-accent-500/10 px-3 py-1 text-xs font-medium text-accent-400">
          {label}
        </span>
      )}
      <h2 className="text-3xl font-bold tracking-tight text-white md:text-4xl lg:text-5xl">
        {title}
      </h2>
      {description && (
        <p className="mx-auto mt-4 max-w-2xl text-base text-neutral-400 md:text-lg">
          {description}
        </p>
      )}
    </div>
  );
}

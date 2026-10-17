"use client";
import { GitBranch, Package } from "lucide-react";
import type { DeployApp } from "@/lib/api";

type Row = { label: string; value: React.ReactNode; mono?: boolean };

/**
 * Where the app comes from and how it runs: repository + branch (or image),
 * framework, port, command, schedule, subdomain. Makes no API call of its own
 * (the custom-domain count lives in the Domains card).
 * Replaces the legacy "App Details" and "Quick Links" cards.
 */
export function SourceCard({ app, wide = false }: { app: DeployApp; wide?: boolean }) {
  const appType = app.app_type ?? "web";
  const isWeb = appType === "web";
  const isImage = app.deploy_source === "image" || !app.repo_url;

  const rows: Row[] = [
    isImage
      ? {
          label: "Image",
          value: (
            <span className="flex items-center gap-1.5 text-neutral-400">
              <Package className="h-3.5 w-3.5 shrink-0" />
              {app.image_url ? (
                <span className="truncate font-mono text-neutral-200">{app.image_url}</span>
              ) : (
                "deployed from an image"
              )}
            </span>
          ),
        }
      : {
          label: "Repository",
          value: (
            <a
              href={app.repo_url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 text-accent-400 hover:underline"
            >
              <GitBranch className="h-3.5 w-3.5" />
              {app.repo_url.replace(/^https?:\/\/github\.com\//, "")}
            </a>
          ),
          mono: true,
        },
    ...(!isImage ? [{ label: "Branch", value: app.branch || "main", mono: true }] : []),
    { label: "Type", value: isWeb ? "Web Service" : appType === "worker" ? "Worker" : "Cron Job" },
    { label: "Framework", value: app.framework || "detecting...", mono: true },
    ...(isWeb ? [{ label: "Port", value: String(app.port || 8080), mono: true }] : []),
    ...(app.command ? [{ label: "Command", value: app.command, mono: true }] : []),
    ...(appType === "cron" && app.cron_schedule
      ? [{ label: "Schedule", value: app.cron_schedule, mono: true }]
      : []),
    ...(isWeb && app.subdomain ? [{ label: "Subdomain", value: app.subdomain, mono: true }] : []),
  ];

  return (
    <section className={`rounded-lg border border-border bg-surface-100 p-5${wide ? " md:col-span-2" : ""}`}>
      <h2 className="mb-4 text-sm font-medium text-white">Source</h2>
      <dl className="space-y-3">
        {rows.map((r) => (
          <div key={r.label} className="flex justify-between gap-4 text-sm">
            <dt className="text-neutral-500">{r.label}</dt>
            <dd className={`min-w-0 truncate text-right text-neutral-200${r.mono ? " font-mono" : ""}`}>
              {r.value}
            </dd>
          </div>
        ))}
      </dl>
    </section>
  );
}

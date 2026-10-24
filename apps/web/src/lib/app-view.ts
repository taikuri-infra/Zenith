/**
 * Display helpers shared by the Overview and Apps pages. Pure functions only,
 * so they can be unit-tested without rendering anything.
 */
import type { DeployApp } from "./api";

export type Tone = "ok" | "warn" | "bad" | "idle";

export function statusTone(status: string): Tone {
  switch (status) {
    case "running":
    case "ready":
    case "active":
      return "ok";
    case "building":
    case "deploying":
    case "provisioning":
    case "creating":
    case "pending":
      return "warn";
    case "failed":
    case "error":
    case "crashed":
      return "bad";
    default:
      return "idle";
  }
}

export const toneText: Record<Tone, string> = {
  ok: "text-emerald-400",
  warn: "text-amber-400",
  bad: "text-red-400",
  idle: "text-neutral-500",
};

export const toneDot: Record<Tone, string> = {
  ok: "bg-emerald-400",
  warn: "bg-amber-400 animate-pulse",
  bad: "bg-red-400",
  idle: "bg-neutral-600",
};

const FRAMEWORKS: Record<string, string> = {
  nextjs: "Next.js",
  go: "Go",
  python: "Python",
  django: "Django",
  flask: "Flask",
  rails: "Rails",
  express: "Express",
  static: "Static",
  dockerfile: "Dockerfile",
};

/** Human label for the framework; "unknown" is hidden rather than shown. */
export function frameworkLabel(framework: string | undefined): string | null {
  if (!framework || framework === "unknown") return null;
  return FRAMEWORKS[framework] ?? framework;
}

/** Where the app comes from: an image reference or a repo and branch. */
export function sourceLabel(app: Pick<DeployApp, "deploy_source" | "image_url" | "repo_url" | "branch">): string {
  if (app.deploy_source === "image" || (!app.repo_url && app.image_url)) {
    return (app.image_url ?? "image").replace(/^docker\.io\//, "").replace(/^library\//, "");
  }
  const repo = (app.repo_url ?? "").replace(/^https?:\/\//, "").replace(/\.git$/, "").replace(/^github\.com\//, "");
  return app.branch ? `${repo} @ ${app.branch}` : repo;
}

export function sourceKind(app: Pick<DeployApp, "deploy_source" | "repo_url">): "image" | "git" {
  return app.deploy_source === "image" || !app.repo_url ? "image" : "git";
}

/** "3 min ago", "2 h ago", "5 d ago"; "just now" under a minute. */
export function timeAgo(iso: string | undefined, now: number = Date.now()): string {
  if (!iso) return "—";
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return "—";
  const sec = Math.max(0, Math.round((now - t) / 1000));
  if (sec < 60) return "just now";
  const min = Math.round(sec / 60);
  if (min < 60) return `${min} min ago`;
  const hr = Math.round(min / 60);
  if (hr < 24) return `${hr} h ago`;
  return `${Math.round(hr / 24)} d ago`;
}

export function formatMemory(mb: number | undefined): string {
  if (mb === undefined || Number.isNaN(mb)) return "—";
  return mb >= 1024 ? `${(mb / 1024).toFixed(1)} GB` : `${Math.round(mb)} MB`;
}

export function formatPercent(p: number | undefined): string {
  if (p === undefined || Number.isNaN(p)) return "—";
  return `${p < 10 ? p.toFixed(1) : Math.round(p)}%`;
}

/** Short form of a commit sha or image tag for lists. */
export function shortRef(ref: string | undefined): string {
  if (!ref) return "—";
  if (/^[0-9a-f]{40}$/i.test(ref)) return ref.slice(0, 7);
  const tail = ref.replace(/^docker\.io\//, "");
  return tail.length > 28 ? `${tail.slice(0, 27)}…` : tail;
}

export function appTypeLabel(t: string | undefined): string {
  return t === "worker" ? "Worker" : t === "cron" ? "Cron" : "Web";
}

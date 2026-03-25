"use client";

import { Shell } from "@/components/shell";
import { useProject, useProjectContext } from "@/hooks/use-project";
import { getApi } from "@/lib/get-api";
import { useState, useEffect } from "react";
import { Copy, Check, FileCode2, Terminal, Eye, EyeOff, AlertTriangle } from "lucide-react";
import type { RegistryCredentials } from "@/lib/api";

const FRAMEWORKS = [
  { id: "go", label: "Go", icon: "🐹" },
  { id: "nextjs", label: "Next.js", icon: "▲" },
  { id: "nodejs", label: "Node.js", icon: "🟢" },
  { id: "python", label: "Python", icon: "🐍" },
  { id: "rust", label: "Rust", icon: "🦀" },
];

export default function CIPage() {
  const projectId = useProject();
  const { currentProject } = useProjectContext();
  const api = getApi();

  const [framework, setFramework] = useState("go");
  const [template, setTemplate] = useState<string>("");
  const [copied, setCopied] = useState<string | null>(null);
  const [loadingTemplate, setLoadingTemplate] = useState(false);
  const [creds, setCreds] = useState<RegistryCredentials | null>(null);
  const [showPassword, setShowPassword] = useState(false);

  const projectSlug = currentProject?.slug || "";

  // Load CI template
  useEffect(() => {
    if (!projectSlug) return;
    setLoadingTemplate(true);
    api.ciTemplates
      .get(framework, projectSlug, "app")
      .then((text) => setTemplate(text))
      .catch(() => setTemplate("# Failed to load template\n# Make sure you are logged in and have a project selected."))
      .finally(() => setLoadingTemplate(false));
  }, [framework, projectSlug]);

  // Load registry credentials
  useEffect(() => {
    if (!projectId) return;
    api.registryCredentials
      .get(projectId)
      .then(setCreds)
      .catch(() => setCreds(null));
  }, [projectId]);

  const copy = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 2000);
  };

  const CopyBtn = ({ text, id }: { text: string; id: string }) => (
    <button
      onClick={() => copy(text, id)}
      className="shrink-0 rounded p-1 text-neutral-400 hover:bg-neutral-700 hover:text-white transition-colors"
    >
      {copied === id ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
    </button>
  );

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-xl font-semibold text-white">CI/CD Setup</h1>
          <p className="text-sm text-neutral-400 mt-1">
            Add a GitHub Actions workflow to automatically build and push your image on every commit.
          </p>
        </div>

        {/* No project selected */}
        {!projectId && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-400">
            <AlertTriangle className="inline h-4 w-4 mr-2" />
            Select a project from the sidebar to see your credentials and personalized templates.
          </div>
        )}

        {/* Registry credentials */}
        <div className="rounded-lg border border-border bg-surface-50 p-4 space-y-3">
          <h3 className="text-sm font-semibold text-white flex items-center gap-2">
            <Terminal className="h-4 w-4 text-accent-400" />
            Registry Credentials
          </h3>
          <p className="text-xs text-neutral-500">
            Add these as secrets in your GitHub repository:
            <span className="ml-1 font-mono text-neutral-400">Settings → Secrets → Actions</span>
          </p>

          {creds === null && projectId && (
            <p className="text-xs text-neutral-500">Loading credentials...</p>
          )}

          {creds !== null && !creds.available && (
            <div className="rounded border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-400">
              <AlertTriangle className="inline h-3.5 w-3.5 mr-1.5" />
              {creds.message || "Registry credentials not available. Contact support."}
            </div>
          )}

          {creds?.available && (
            <div className="space-y-2">
              {/* ZENITH_REGISTRY_USER */}
              <div className="flex items-center justify-between rounded-lg bg-surface-200 border border-border px-3 py-2">
                <div className="min-w-0 flex-1">
                  <span className="text-xs font-mono font-medium text-neutral-300">ZENITH_REGISTRY_USER</span>
                  <span className="ml-2 font-mono text-xs text-accent-400">{creds.username}</span>
                </div>
                <CopyBtn text={creds.username} id="reg-user" />
              </div>

              {/* ZENITH_REGISTRY_PASS */}
              <div className="flex items-center justify-between rounded-lg bg-surface-200 border border-border px-3 py-2">
                <div className="min-w-0 flex-1">
                  <span className="text-xs font-mono font-medium text-neutral-300">ZENITH_REGISTRY_PASS</span>
                  <span className="ml-2 font-mono text-xs text-neutral-400">
                    {showPassword ? creds.password : "••••••••••••••••"}
                  </span>
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  <button
                    onClick={() => setShowPassword((v) => !v)}
                    className="rounded p-1 text-neutral-400 hover:text-white transition-colors"
                    title={showPassword ? "Hide" : "Reveal"}
                  >
                    {showPassword ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                  </button>
                  <CopyBtn text={creds.password} id="reg-pass" />
                </div>
              </div>

              {/* Push prefix info */}
              <div className="rounded-lg bg-surface-200 border border-border px-3 py-2">
                <p className="text-[10px] text-neutral-500 mb-1 uppercase tracking-wider font-medium">Your push path</p>
                <div className="flex items-center gap-2">
                  <code className="flex-1 font-mono text-xs text-neutral-300">{creds.push_prefix}/&lt;service&gt;:latest</code>
                  <CopyBtn text={creds.push_prefix} id="push-prefix" />
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Framework selector */}
        <div>
          <p className="text-xs font-medium text-neutral-500 uppercase tracking-wider mb-2">Framework</p>
          <div className="flex gap-2 flex-wrap">
            {FRAMEWORKS.map((fw) => (
              <button
                key={fw.id}
                onClick={() => setFramework(fw.id)}
                className={`flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors ${
                  framework === fw.id
                    ? "bg-accent-500 text-white"
                    : "border border-border bg-surface-200 text-neutral-400 hover:text-white"
                }`}
              >
                <span>{fw.icon}</span>
                {fw.label}
              </button>
            ))}
          </div>
        </div>

        {/* Workflow template */}
        <div className="rounded-lg border border-border bg-surface-50 overflow-hidden">
          <div className="flex items-center justify-between border-b border-border px-4 py-2.5 bg-surface-100">
            <div className="flex items-center gap-2 text-xs text-neutral-400">
              <FileCode2 className="h-3.5 w-3.5" />
              <span className="font-mono">.github/workflows/zenith-deploy.yml</span>
            </div>
            <CopyBtn text={template} id="template" />
          </div>
          <pre className="overflow-x-auto p-4 text-xs text-neutral-300 max-h-[400px] leading-5 font-mono">
            {loadingTemplate ? (
              <span className="text-neutral-600">Loading template...</span>
            ) : (
              template
            )}
          </pre>
        </div>

        {/* Quick steps */}
        <div className="rounded-lg border border-border bg-surface-50 p-4 space-y-3">
          <h3 className="text-sm font-semibold text-white">Quick Setup Steps</h3>
          <ol className="space-y-2 text-sm text-neutral-400 list-none">
            {[
              "Copy the workflow file above into .github/workflows/zenith-deploy.yml in your repo",
              "Add ZENITH_REGISTRY_USER and ZENITH_REGISTRY_PASS to your GitHub repo secrets",
              "Push a commit to main — GitHub Actions will build and push your image automatically",
              "Come back here and deploy the project using your new image",
            ].map((step, i) => (
              <li key={i} className="flex items-start gap-3">
                <span className="shrink-0 flex h-5 w-5 items-center justify-center rounded-full bg-accent-500/20 text-[10px] font-bold text-accent-400 mt-0.5">
                  {i + 1}
                </span>
                {step}
              </li>
            ))}
          </ol>
        </div>
      </div>
    </Shell>
  );
}

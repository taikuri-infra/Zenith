"use client";

import { Shell } from "@/components/shell";
import { useProject, useProjectContext } from "@/hooks/use-project";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { useState, useEffect } from "react";
import { Copy, Check, FileCode2, Terminal } from "lucide-react";

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
  const [framework, setFramework] = useState("go");
  const [template, setTemplate] = useState<string>("");
  const [copied, setCopied] = useState<string | null>(null);
  const [loadingTemplate, setLoadingTemplate] = useState(false);

  const projectSlug = currentProject?.slug || "<your-project>";

  useEffect(() => {
    setLoadingTemplate(true);
    const api = getApi();
    // CI template endpoint returns text, handle it
    fetch(`/api/v1/ci-templates/${framework}?project=${projectSlug}&service=app`, {
      headers: { Authorization: `Bearer ${typeof window !== "undefined" ? localStorage.getItem("zenith_access_token") || "" : ""}` },
    })
      .then(async (res) => {
        if (res.ok) {
          setTemplate(await res.text());
        } else {
          setTemplate("# Failed to load template");
        }
      })
      .catch(() => setTemplate("# Failed to load template"))
      .finally(() => setLoadingTemplate(false));
  }, [framework, projectSlug]);

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 2000);
  };

  const CopyButton = ({ text, id }: { text: string; id: string }) => (
    <button
      onClick={() => copyToClipboard(text, id)}
      className="rounded p-1 text-zinc-400 hover:bg-zinc-700 hover:text-white"
    >
      {copied === id ? <Check className="h-4 w-4 text-green-400" /> : <Copy className="h-4 w-4" />}
    </button>
  );

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-white">CI/CD Setup</h1>
          <p className="text-sm text-zinc-400 mt-1">
            Add a GitHub Actions workflow to automatically build and deploy your app on every push.
          </p>
        </div>

        {/* Framework selector */}
        <div className="flex gap-2">
          {FRAMEWORKS.map((fw) => (
            <button
              key={fw.id}
              onClick={() => setFramework(fw.id)}
              className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition ${
                framework === fw.id
                  ? "bg-blue-600 text-white"
                  : "bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-white"
              }`}
            >
              <span>{fw.icon}</span>
              {fw.label}
            </button>
          ))}
        </div>

        {/* Template */}
        <div className="rounded-lg border border-zinc-700 bg-zinc-900">
          <div className="flex items-center justify-between border-b border-zinc-700 px-4 py-2">
            <div className="flex items-center gap-2 text-sm text-zinc-400">
              <FileCode2 className="h-4 w-4" />
              .github/workflows/zenith-deploy.yml
            </div>
            <CopyButton text={template} id="template" />
          </div>
          <pre className="overflow-x-auto p-4 text-sm text-zinc-300 max-h-96">
            {loadingTemplate ? "Loading..." : template}
          </pre>
        </div>

        {/* Secrets to add */}
        <div className="rounded-lg border border-zinc-700 bg-zinc-800/50 p-4 space-y-3">
          <h3 className="text-sm font-semibold text-white flex items-center gap-2">
            <Terminal className="h-4 w-4" />
            Secrets to add to your GitHub repository
          </h3>
          <p className="text-xs text-zinc-500">
            Go to your repo Settings &rarr; Secrets &rarr; Actions and add these:
          </p>
          <div className="space-y-2">
            {[
              { name: "ZENITH_REGISTRY_USER", value: currentProject?.harbor_robot_user || "<robot-user>", secret: false },
              { name: "ZENITH_REGISTRY_PASS", value: "••••••••", secret: true },
            ].map((s) => (
              <div
                key={s.name}
                className="flex items-center justify-between rounded bg-zinc-900 px-3 py-2"
              >
                <div>
                  <span className="text-sm font-mono text-zinc-300">{s.name}</span>
                  <span className="ml-2 text-xs text-zinc-500">
                    {s.secret ? "(from project credentials)" : `= ${s.value}`}
                  </span>
                </div>
                {!s.secret && <CopyButton text={s.value} id={s.name} />}
              </div>
            ))}
          </div>
        </div>
      </div>
    </Shell>
  );
}

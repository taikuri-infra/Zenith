"use client";

import { Shell } from "@/components/shell";
import { useToast } from "@/components/toast";
import { getApi } from "@/lib/get-api";
import type { Environment } from "@/lib/api";
import { useParams } from "next/navigation";
import { useState, useEffect, useCallback } from "react";
import {
  Globe,
  TestTube2,
  CheckCircle2,
  AlertCircle,
  Loader2,
  ExternalLink,
} from "lucide-react";

export default function EnvironmentsPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { toast } = useToast();
  const api = getApi();

  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchEnvironments = useCallback(async () => {
    try {
      const resp = await api.environments.list(projectId);
      setEnvironments(resp.environments || []);
    } catch {
      toast("error", "Failed to load environments");
    } finally {
      setLoading(false);
    }
  }, [projectId, api, toast]);

  useEffect(() => {
    fetchEnvironments();
  }, [fetchEnvironments]);

  const statusIcon = (status: string) => {
    switch (status) {
      case "active":
        return <CheckCircle2 className="h-4 w-4 text-green-400" />;
      case "provisioning":
        return <Loader2 className="h-4 w-4 text-amber-400 animate-spin" />;
      case "error":
        return <AlertCircle className="h-4 w-4 text-red-400" />;
      default:
        return null;
    }
  };

  const envIcon = (name: string) => {
    return name === "production" ? (
      <Globe className="h-5 w-5 text-blue-400" />
    ) : (
      <TestTube2 className="h-5 w-5 text-amber-400" />
    );
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Environments</h1>
          <p className="text-sm text-neutral-500">
            Your project has separate environments for production and staging
          </p>
        </div>

        {loading ? (
          <div className="text-sm text-neutral-500">Loading...</div>
        ) : environments.length === 0 ? (
          <div className="text-center py-12 text-neutral-500">
            <Globe className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No environments</p>
            <p className="text-xs mt-1">
              Environments are created automatically when you create a project
            </p>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            {environments.map((env) => (
              <div
                key={env.id}
                className={`rounded-lg border p-5 space-y-3 ${
                  env.name === "production"
                    ? "border-blue-500/30 bg-blue-500/5"
                    : "border-amber-500/30 bg-amber-500/5"
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    {envIcon(env.name)}
                    <div>
                      <h3 className="text-sm font-medium text-white capitalize">
                        {env.name}
                      </h3>
                      <p className="text-xs text-neutral-500">
                        {env.is_default ? "Default environment" : "Dev/testing"}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-1.5">
                    {statusIcon(env.status)}
                    <span className="text-xs text-neutral-400 capitalize">
                      {env.status}
                    </span>
                  </div>
                </div>

                <div className="space-y-2 text-xs text-neutral-500">
                  <div className="flex items-center justify-between">
                    <span>URL pattern</span>
                    <code className="text-neutral-400">
                      {env.name === "production"
                        ? "*.apps.freezenith.com"
                        : "*.dev.apps.freezenith.com"}
                    </code>
                  </div>
                  <div className="flex items-center justify-between">
                    <span>Resources</span>
                    <span className="text-neutral-400">
                      {env.name === "production"
                        ? "Per plan limits"
                        : "Minimal (0.25 CPU, 256MB)"}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span>Created</span>
                    <span className="text-neutral-400">
                      {new Date(env.created_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>

                <div className="pt-2 border-t border-border/50">
                  <a
                    href={`/projects/${projectId}?env=${env.slug}`}
                    className="flex items-center gap-1 text-xs text-accent-400 hover:text-accent-300"
                  >
                    View services
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Info box */}
        <div className="rounded-lg border border-border bg-card p-4 space-y-2">
          <h3 className="text-sm font-medium text-white">
            How environments work
          </h3>
          <ul className="text-xs text-neutral-500 space-y-1 list-disc list-inside">
            <li>
              <strong>Production</strong> — Your live environment with full plan
              resources
            </li>
            <li>
              <strong>Staging</strong> — Free dev environment with minimal
              resources (Pro+ plans)
            </li>
            <li>
              Use <code>zen dev</code> CLI to connect your local code to staging
              services
            </li>
            <li>
              Deploy to staging via GitHub Actions with{" "}
              <code>dotechhq/zenith-stage@v1</code>
            </li>
          </ul>
        </div>
      </div>
    </Shell>
  );
}

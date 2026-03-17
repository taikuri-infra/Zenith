"use client";

import { Shell } from "@/components/shell";
import { useToast } from "@/components/toast";
import { getApi } from "@/lib/get-api";
import type {
  ComposeImportResult,
  ManagedService,
  ParsedService,
  ParsedManaged,
} from "@/lib/api";
import { useRouter } from "next/navigation";
import { useState, useCallback, useRef } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Upload,
  Check,
  Loader2,
  Copy,
  AlertTriangle,
  Database,
  Server,
  Rocket,
} from "lucide-react";

type Step = 1 | 2 | 3;

export default function NewProjectPage() {
  const router = useRouter();
  const { toast } = useToast();
  const api = getApi();

  // Step state
  const [step, setStep] = useState<Step>(1);

  // Step 1: Paste & Name
  const [name, setName] = useState("");
  const [composeContent, setComposeContent] = useState("");
  const [parsing, setParsing] = useState(false);

  // Step 2: Review & Push
  const [projectId, setProjectId] = useState("");
  const [parseResult, setParseResult] = useState<ComposeImportResult | null>(null);
  const [provisionedServices, setProvisionedServices] = useState<ManagedService[]>([]);
  const [provisioning, setProvisioning] = useState(false);

  // Step 3: Deploy
  const [deploying, setDeploying] = useState(false);
  const [deployDone, setDeployDone] = useState(false);
  const [deployStatus, setDeployStatus] = useState<Record<string, "pending" | "deploying" | "done" | "error">>({});

  const fileInputRef = useRef<HTMLInputElement>(null);

  // Step 1 → Step 2: Create project + parse compose
  const handleStep1Next = useCallback(async () => {
    if (!name.trim()) {
      toast("error", "Project name is required");
      return;
    }
    if (!composeContent.trim()) {
      toast("error", "Docker Compose content is required");
      return;
    }

    setParsing(true);
    try {
      // Create project via API directly (not context — avoids provider dependency)
      const project = await api.projects.create({ name: name.trim() });
      setProjectId(project.id);

      // Parse compose
      const result = await api.composeImport.parse(project.id, composeContent);
      setParseResult(result);

      if (!result.valid) {
        toast("error", "Compose file has errors. Fix them and try again.");
        setParsing(false);
        return;
      }

      // Auto-provision managed services
      if (result.managed_services.length > 0) {
        setProvisioning(true);
        const provisioned: ManagedService[] = [];
        for (const ms of result.managed_services) {
          try {
            const svc = await api.managedServices.provision(project.id, {
              service_type: ms.type,
              name: ms.name,
              version: ms.version,
            });
            provisioned.push(svc);
          } catch (e) {
            toast("error", `Failed to provision ${ms.name}: ${e}`);
          }
        }
        setProvisionedServices(provisioned);
        setProvisioning(false);
      }

      setStep(2);
    } catch (e) {
      toast("error", `Failed: ${e}`);
    } finally {
      setParsing(false);
    }
  }, [name, composeContent, api, toast]);

  // File upload handler
  const handleFileUpload = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = (ev) => {
        setComposeContent(ev.target?.result as string);
      };
      reader.readAsText(file);
    },
    []
  );

  // Step 3: Deploy each parsed service via the real API
  const handleDeploy = useCallback(async () => {
    if (!parseResult || !projectId) return;
    setDeploying(true);

    // Initialize status for all services
    const status: Record<string, "pending" | "deploying" | "done" | "error"> = {};
    for (const svc of parseResult.services) {
      status[svc.name] = "pending";
    }
    setDeployStatus({ ...status });

    let allOk = true;

    for (const svc of parseResult.services) {
      status[svc.name] = "deploying";
      setDeployStatus({ ...status });

      try {
        // Determine image URL
        let imageUrl = svc.image || "";
        if (svc.build_context && !imageUrl) {
          // User must push their image; use the registry path shown in Step 2
          imageUrl = `registry.stage.freezenith.com/${projectId}/${svc.name}:latest`;
        }

        // Collect env vars from compose translation
        const envVars = svc.env_vars
          .filter((ev) => ev.zenith)
          .map((ev) => ({ key: ev.key, value: ev.zenith }));

        await api.appsDeploy.create({
          project_id: projectId,
          name: svc.name,
          deploy_source: "image",
          image_url: imageUrl,
          port: svc.port || 8080,
          app_type: svc.is_public ? "web" : "worker",
          exposure: svc.is_public ? "public" : "public",
          env_vars: envVars.length > 0 ? envVars : undefined,
        });

        status[svc.name] = "done";
        setDeployStatus({ ...status });
      } catch (e) {
        status[svc.name] = "error";
        setDeployStatus({ ...status });
        toast("error", `Failed to deploy ${svc.name}: ${e}`);
        allOk = false;
      }
    }

    if (allOk) {
      setDeployDone(true);
      toast("success", "All services deployed successfully!");
    }
    setDeploying(false);
  }, [parseResult, projectId, api, toast]);

  return (
    <Shell>
      <div className="mx-auto max-w-3xl space-y-8">
        {/* Progress bar */}
        <div className="flex items-center gap-2">
          <button onClick={() => router.back()} className="text-neutral-400 hover:text-white">
            <ArrowLeft className="h-5 w-5" />
          </button>
          <h1 className="text-lg font-semibold text-white">Deploy Project</h1>
          <div className="ml-auto flex items-center gap-2 text-sm text-neutral-400">
            <StepDot active={step >= 1} done={step > 1} label="1" />
            <div className="h-px w-6 bg-neutral-700" />
            <StepDot active={step >= 2} done={step > 2} label="2" />
            <div className="h-px w-6 bg-neutral-700" />
            <StepDot active={step >= 3} done={deployDone} label="3" />
          </div>
        </div>

        {/* Step 1: Paste & Name */}
        {step === 1 && (
          <div className="space-y-6">
            <div>
              <h2 className="text-base font-medium text-white">Step 1: Paste & Name</h2>
              <p className="mt-1 text-sm text-neutral-400">
                Give your project a name and paste your docker-compose.yml
              </p>
            </div>

            <div>
              <label className="mb-1.5 block text-sm font-medium text-neutral-300">
                Project Name
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="My SaaS"
                className="w-full rounded-lg border border-border bg-surface-200 px-4 py-2.5 text-sm text-white placeholder:text-neutral-500 focus:border-brand focus:outline-none"
              />
            </div>

            <div>
              <div className="mb-1.5 flex items-center justify-between">
                <label className="text-sm font-medium text-neutral-300">
                  docker-compose.yml
                </label>
                <button
                  onClick={() => fileInputRef.current?.click()}
                  className="flex items-center gap-1.5 text-xs text-brand hover:text-brand/80"
                >
                  <Upload className="h-3.5 w-3.5" />
                  Upload file
                </button>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".yml,.yaml"
                  onChange={handleFileUpload}
                  className="hidden"
                />
              </div>
              <textarea
                value={composeContent}
                onChange={(e) => setComposeContent(e.target.value)}
                placeholder={`version: "3.8"\nservices:\n  api:\n    build: ./api\n    ports:\n      - "8080:8080"\n  db:\n    image: postgres:16`}
                rows={14}
                className="w-full rounded-lg border border-border bg-surface-200 px-4 py-3 font-mono text-xs text-white placeholder:text-neutral-600 focus:border-brand focus:outline-none"
              />
            </div>

            <div className="flex justify-end">
              <button
                onClick={handleStep1Next}
                disabled={parsing}
                className="flex items-center gap-2 rounded-lg bg-brand px-5 py-2.5 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-50"
              >
                {parsing ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <ArrowRight className="h-4 w-4" />
                )}
                {parsing ? "Analyzing..." : "Continue"}
              </button>
            </div>
          </div>
        )}

        {/* Step 2: Review & Push */}
        {step === 2 && parseResult && (
          <div className="space-y-6">
            <div>
              <h2 className="text-base font-medium text-white">Step 2: Review & Push</h2>
              <p className="mt-1 text-sm text-neutral-400">
                We detected your services. Push your images when ready.
              </p>
            </div>

            {/* Warnings */}
            {parseResult.warnings.length > 0 && (
              <div className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 p-4">
                <div className="flex items-center gap-2 text-sm font-medium text-yellow-400">
                  <AlertTriangle className="h-4 w-4" />
                  Warnings
                </div>
                <ul className="mt-2 space-y-1 text-xs text-yellow-300/80">
                  {parseResult.warnings.map((w, i) => (
                    <li key={i}>• {w}</li>
                  ))}
                </ul>
              </div>
            )}

            {/* App services */}
            <div>
              <h3 className="mb-3 flex items-center gap-2 text-sm font-medium text-neutral-300">
                <Server className="h-4 w-4" />
                App Services ({parseResult.services.length})
              </h3>
              <div className="space-y-3">
                {parseResult.services.map((svc) => (
                  <ServiceCard key={svc.name} service={svc} projectId={projectId} />
                ))}
              </div>
            </div>

            {/* Managed services */}
            {parseResult.managed_services.length > 0 && (
              <div>
                <h3 className="mb-3 flex items-center gap-2 text-sm font-medium text-neutral-300">
                  <Database className="h-4 w-4" />
                  Managed Services ({parseResult.managed_services.length})
                </h3>
                <div className="space-y-3">
                  {parseResult.managed_services.map((ms) => (
                    <ManagedServiceCard
                      key={ms.name}
                      managed={ms}
                      provisioned={provisionedServices.find((p) => p.name === ms.name)}
                      provisioning={provisioning}
                    />
                  ))}
                </div>
              </div>
            )}

            {/* Env vars preview */}
            {parseResult.services.some((s) => s.env_vars.length > 0) && (
              <div>
                <h3 className="mb-3 text-sm font-medium text-neutral-300">
                  Environment Variables (auto-translated)
                </h3>
                <div className="rounded-lg border border-border bg-surface-200 p-4">
                  <div className="space-y-2">
                    {parseResult.services.flatMap((svc) =>
                      svc.env_vars
                        .filter((ev) => ev.original !== ev.zenith)
                        .map((ev) => (
                          <div key={`${svc.name}-${ev.key}`} className="text-xs">
                            <span className="text-neutral-400">{svc.name}/</span>
                            <span className="font-medium text-white">{ev.key}</span>
                            <div className="mt-0.5 flex items-center gap-2 text-neutral-500">
                              <span className="line-through">{ev.original}</span>
                              <ArrowRight className="h-3 w-3" />
                              <span className="text-emerald-400">{ev.zenith}</span>
                            </div>
                          </div>
                        ))
                    )}
                  </div>
                </div>
              </div>
            )}

            <div className="flex justify-between">
              <button
                onClick={() => setStep(1)}
                className="flex items-center gap-2 rounded-lg border border-border px-4 py-2.5 text-sm text-neutral-300 hover:text-white"
              >
                <ArrowLeft className="h-4 w-4" />
                Back
              </button>
              <button
                onClick={() => setStep(3)}
                className="flex items-center gap-2 rounded-lg bg-brand px-5 py-2.5 text-sm font-medium text-white hover:bg-brand/90"
              >
                <Rocket className="h-4 w-4" />
                Deploy
              </button>
            </div>
          </div>
        )}

        {/* Step 3: Deploy */}
        {step === 3 && (
          <div className="space-y-6">
            <div>
              <h2 className="text-base font-medium text-white">Step 3: Deploy</h2>
              <p className="mt-1 text-sm text-neutral-400">
                {deployDone
                  ? "All services are running!"
                  : "Click deploy to launch all services."}
              </p>
            </div>

            {/* Deploy status per service */}
            {parseResult && (
              <div className="space-y-3">
                {parseResult.services.map((svc) => (
                  <div
                    key={svc.name}
                    className="flex items-center justify-between rounded-lg border border-border bg-surface-200 px-4 py-3"
                  >
                    <div className="flex items-center gap-3">
                      <Server className="h-4 w-4 text-neutral-400" />
                      <span className="text-sm font-medium text-white">{svc.name}</span>
                      {svc.port > 0 && (
                        <span className="text-xs text-neutral-500">:{svc.port}</span>
                      )}
                    </div>
                    {deployStatus[svc.name] === "done" ? (
                      <span className="flex items-center gap-1.5 text-xs text-emerald-400">
                        <Check className="h-3.5 w-3.5" />
                        Deployed
                      </span>
                    ) : deployStatus[svc.name] === "deploying" ? (
                      <Loader2 className="h-4 w-4 animate-spin text-brand" />
                    ) : deployStatus[svc.name] === "error" ? (
                      <span className="flex items-center gap-1.5 text-xs text-red-400">
                        <AlertTriangle className="h-3.5 w-3.5" />
                        Failed
                      </span>
                    ) : (
                      <span className="text-xs text-neutral-500">Pending</span>
                    )}
                  </div>
                ))}
              </div>
            )}

            {deployDone && parseResult && (
              <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-4">
                <p className="text-sm font-medium text-emerald-400">
                  Your project is live!
                </p>
                <div className="mt-2 space-y-1">
                  {parseResult.services
                    .filter((s) => s.url)
                    .map((s) => (
                      <a
                        key={s.name}
                        href={s.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="block text-xs text-emerald-300 underline hover:text-emerald-200"
                      >
                        {s.url}
                      </a>
                    ))}
                </div>
              </div>
            )}

            <div className="flex justify-between">
              {!deployDone && (
                <button
                  onClick={() => setStep(2)}
                  className="flex items-center gap-2 rounded-lg border border-border px-4 py-2.5 text-sm text-neutral-300 hover:text-white"
                >
                  <ArrowLeft className="h-4 w-4" />
                  Back
                </button>
              )}
              <div className="ml-auto">
                {deployDone ? (
                  <button
                    onClick={() => router.push("/")}
                    className="flex items-center gap-2 rounded-lg bg-brand px-5 py-2.5 text-sm font-medium text-white hover:bg-brand/90"
                  >
                    Go to Dashboard
                    <ArrowRight className="h-4 w-4" />
                  </button>
                ) : (
                  <button
                    onClick={handleDeploy}
                    disabled={deploying}
                    className="flex items-center gap-2 rounded-lg bg-brand px-5 py-2.5 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-50"
                  >
                    {deploying ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Rocket className="h-4 w-4" />
                    )}
                    {deploying ? "Deploying..." : "Deploy All"}
                  </button>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </Shell>
  );
}

// Sub-components

function StepDot({
  active,
  done,
  label,
}: {
  active: boolean;
  done: boolean;
  label: string;
}) {
  return (
    <div
      className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-medium ${
        done
          ? "bg-emerald-500 text-white"
          : active
          ? "bg-brand text-white"
          : "bg-surface-200 text-neutral-500"
      }`}
    >
      {done ? <Check className="h-3.5 w-3.5" /> : label}
    </div>
  );
}

function ServiceCard({
  service,
  projectId,
}: {
  service: ParsedService;
  projectId: string;
}) {
  const [copied, setCopied] = useState(false);

  const pushCmd = `docker push registry.stage.freezenith.com/${projectId}/${service.name}:latest`;
  const handleCopy = () => {
    navigator.clipboard.writeText(pushCmd);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="rounded-lg border border-border bg-surface-200 p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Server className="h-4 w-4 text-brand" />
          <span className="text-sm font-medium text-white">{service.name}</span>
          {service.port > 0 && (
            <span className="rounded bg-neutral-700 px-1.5 py-0.5 text-[10px] text-neutral-300">
              :{service.port}
            </span>
          )}
          {service.is_public && (
            <span className="rounded bg-emerald-500/20 px-1.5 py-0.5 text-[10px] text-emerald-400">
              public
            </span>
          )}
        </div>
        {service.url && (
          <span className="text-xs text-neutral-400">{service.url}</span>
        )}
      </div>
      {service.build_context && (
        <div className="mt-3">
          <p className="mb-1 text-[11px] text-neutral-500">Push command:</p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded bg-neutral-900 px-3 py-2 font-mono text-[11px] text-neutral-300">
              {pushCmd}
            </code>
            <button
              onClick={handleCopy}
              className="shrink-0 rounded border border-border p-1.5 text-neutral-400 hover:text-white"
            >
              {copied ? (
                <Check className="h-3.5 w-3.5 text-emerald-400" />
              ) : (
                <Copy className="h-3.5 w-3.5" />
              )}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function ManagedServiceCard({
  managed,
  provisioned,
  provisioning,
}: {
  managed: ParsedManaged;
  provisioned?: ManagedService;
  provisioning: boolean;
}) {
  const typeIcon = managed.type === "postgresql" ? "P" : "R";
  const typeColor =
    managed.type === "postgresql"
      ? "bg-blue-500/20 text-blue-400"
      : "bg-red-500/20 text-red-400";

  return (
    <div className="flex items-center justify-between rounded-lg border border-border bg-surface-200 px-4 py-3">
      <div className="flex items-center gap-3">
        <span
          className={`flex h-6 w-6 items-center justify-center rounded text-[10px] font-bold ${typeColor}`}
        >
          {typeIcon}
        </span>
        <div>
          <span className="text-sm font-medium text-white">{managed.name}</span>
          <span className="ml-2 text-xs text-neutral-500">
            {managed.type} {managed.version}
          </span>
        </div>
      </div>
      <div>
        {provisioned ? (
          <span className="flex items-center gap-1.5 text-xs text-emerald-400">
            <Check className="h-3.5 w-3.5" />
            {provisioned.status}
          </span>
        ) : provisioning ? (
          <span className="flex items-center gap-1.5 text-xs text-brand">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Provisioning...
          </span>
        ) : (
          <span className="text-xs text-neutral-500">Detected</span>
        )}
      </div>
    </div>
  );
}

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
  AlignLeft,
  Sparkles,
  XCircle,
  Globe,
  Lock,
  Shield,
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
  const [deployPhase, setDeployPhase] = useState<"managed" | "backend" | "frontend" | "done">("managed");
  const [deployStatus, setDeployStatus] = useState<Record<string, { state: "pending" | "creating" | "waiting" | "running" | "error"; url?: string; error?: string }>>({});
  const [deployLog, setDeployLog] = useState<string[]>([]);

  // Parse errors & AI suggestions (shown inline on step 1)
  const [parseErrors, setParseErrors] = useState<string[]>([]);
  const [aiSuggestions, setAISuggestions] = useState<string[]>([]);

  // Exposure overrides: user can toggle backend services to public
  const [exposureOverrides, setExposureOverrides] = useState<Record<string, boolean>>({});

  const fileInputRef = useRef<HTMLInputElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const lineNumbersRef = useRef<HTMLDivElement>(null);

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
    setParseErrors([]);
    setAISuggestions([]);
    try {
      // Create project (or reuse existing)
      let pid = projectId;
      if (!pid) {
        const project = await api.projects.create({ name: name.trim() });
        pid = project.id;
        setProjectId(pid);
      }

      // Parse compose
      const result = await api.composeImport.parse(pid, composeContent);
      setParseResult(result);

      if (!result.valid) {
        setParseErrors(result.errors || []);
        setAISuggestions(result.ai_suggestions || []);
        setParsing(false);
        return;
      }

      // AI suggestions only shown when there are errors (not generic tips on success)

      // Auto-provision managed services (only PostgreSQL — Redis not supported yet)
      const provisionable = (result.managed_services || []).filter((ms) => ms.type === "postgresql");
      if (provisionable.length > 0) {
        setProvisioning(true);
        const provisioned: ManagedService[] = [];
        for (const ms of provisionable) {
          try {
            const svc = await api.managedServices.provision(pid, {
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
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      if (msg.includes("409") || msg.toLowerCase().includes("conflict") || msg.toLowerCase().includes("already exists")) {
        toast("error", `A project named "${name.trim()}" already exists. Choose a different name.`);
      } else {
        toast("error", msg);
      }
    } finally {
      setParsing(false);
    }
  }, [name, composeContent, projectId, api, toast]);

  // Format YAML: fix common indentation issues
  const handleFormatYaml = useCallback(() => {
    const lines = composeContent.split("\n");
    // Simple fix: if "services:" is indented, dedent it and everything below
    const formatted = lines.map((line) => {
      // Remove leading spaces from top-level keys (version, services, volumes, networks)
      if (/^\s+(version|services|volumes|networks):/.test(line)) {
        return line.trimStart();
      }
      return line;
    });
    setComposeContent(formatted.join("\n"));
    setParseErrors([]);
    setAISuggestions([]);
    toast("success", "YAML formatted");
  }, [composeContent, toast]);

  // Sync line numbers scroll with textarea
  const handleTextareaScroll = useCallback(() => {
    if (textareaRef.current && lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = textareaRef.current.scrollTop;
    }
  }, []);

  const lineCount = composeContent.split("\n").length || 1;

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

  const addLog = useCallback((msg: string) => {
    setDeployLog((prev) => [...prev, `[${new Date().toLocaleTimeString()}] ${msg}`]);
  }, []);

  // Poll app until running or error (max 60s)
  const waitForApp = useCallback(async (appId: string, appName: string): Promise<{ ok: boolean; url?: string; error?: string }> => {
    const maxAttempts = 20;
    for (let i = 0; i < maxAttempts; i++) {
      await new Promise((r) => setTimeout(r, 3000));
      try {
        const app = await api.appsDeploy.get(appId);
        if (app.status === "running" || app.status === "active") {
          return { ok: true, url: app.url };
        }
        if (app.status === "error" || app.status === "failed" || app.status === "crash_loop") {
          return { ok: false, error: `App entered ${app.status} state` };
        }
        // Still deploying...
      } catch {
        // Ignore transient errors during polling
      }
    }
    return { ok: false, error: "Timed out waiting for app to start (60s)" };
  }, [api]);

  // Step 3: Deploy with phases: managed → backend → frontend
  const handleDeploy = useCallback(async () => {
    if (!parseResult || !projectId) return;
    setDeploying(true);
    setDeployLog([]);

    const slug = name.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
    const services = parseResult.services || [];
    const managed = (parseResult.managed_services || []).filter((ms) => ms.type === "postgresql");

    // Categorize services
    const backends = services.filter((s) => !(exposureOverrides[s.name] ?? s.is_public));
    const frontends = services.filter((s) => exposureOverrides[s.name] ?? s.is_public);

    // Init status
    const status: Record<string, { state: "pending" | "creating" | "waiting" | "running" | "error"; url?: string; error?: string }> = {};
    for (const ms of managed) status[`db:${ms.name}`] = { state: "pending" };
    for (const svc of backends) status[svc.name] = { state: "pending" };
    for (const svc of frontends) status[svc.name] = { state: "pending" };
    setDeployStatus({ ...status });

    let allOk = true;

    // ── Phase 1: Managed Services (Database) ──
    if (managed.length > 0) {
      setDeployPhase("managed");
      addLog(`Phase 1: Provisioning ${managed.length} managed service(s)...`);

      for (const ms of managed) {
        const key = `db:${ms.name}`;
        status[key] = { state: "creating" };
        setDeployStatus({ ...status });
        addLog(`Provisioning ${ms.type} "${ms.name}" v${ms.version}...`);

        // Already provisioned in step 1, just mark done
        const existing = provisionedServices.find((p) => p.name === ms.name);
        if (existing) {
          status[key] = { state: "running" };
          setDeployStatus({ ...status });
          addLog(`✓ ${ms.name} already provisioned (${existing.status})`);
        } else {
          try {
            await api.managedServices.provision(projectId, {
              service_type: ms.type,
              name: ms.name,
              version: ms.version,
            });
            status[key] = { state: "running" };
            setDeployStatus({ ...status });
            addLog(`✓ ${ms.name} provisioned`);
          } catch (e) {
            const msg = e instanceof Error ? e.message : String(e);
            status[key] = { state: "error", error: msg };
            setDeployStatus({ ...status });
            addLog(`✗ ${ms.name} failed: ${msg}`);
            allOk = false;
          }
        }
      }
    }

    // Helper to deploy a service
    const deploySvc = async (svc: ParsedService) => {
      status[svc.name] = { state: "creating" };
      setDeployStatus({ ...status });

      let imageUrl = svc.image || "";
      if (svc.build_context && !imageUrl) {
        imageUrl = `registry.stage.freezenith.com/${projectId}/${svc.name}:latest`;
      }

      const envVars = svc.env_vars
        .filter((ev) => ev.zenith)
        .map((ev) => ({ key: ev.key, value: ev.zenith }));

      const appName = `${slug}-${svc.name}`;
      const isPublic = exposureOverrides[svc.name] ?? svc.is_public;

      addLog(`Creating ${appName} (${isPublic ? "public" : "private"}, port ${svc.port || 8080})...`);

      try {
        const app = await api.appsDeploy.create({
          project_id: projectId,
          name: appName,
          deploy_source: "image",
          image_url: imageUrl,
          port: svc.port || 8080,
          app_type: isPublic ? "web" : "worker",
          exposure: isPublic ? "public" : "protected",
          env_vars: envVars.length > 0 ? envVars : undefined,
        });

        status[svc.name] = { state: "waiting" };
        setDeployStatus({ ...status });
        addLog(`Waiting for ${appName} to start...`);

        const result = await waitForApp(app.id, appName);
        if (result.ok) {
          status[svc.name] = { state: "running", url: result.url };
          setDeployStatus({ ...status });
          addLog(`✓ ${appName} is running${result.url ? ` → ${result.url}` : ""}`);
        } else {
          status[svc.name] = { state: "error", error: result.error };
          setDeployStatus({ ...status });
          addLog(`✗ ${appName} failed: ${result.error}`);
          allOk = false;
        }
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        status[svc.name] = { state: "error", error: msg };
        setDeployStatus({ ...status });
        addLog(`✗ ${appName} failed: ${msg}`);
        allOk = false;
      }
    };

    // ── Phase 2: Backend services ──
    if (backends.length > 0) {
      setDeployPhase("backend");
      addLog(`Phase 2: Deploying ${backends.length} backend service(s)...`);
      for (const svc of backends) {
        await deploySvc(svc);
      }
    }

    // ── Phase 3: Frontend services ──
    if (frontends.length > 0) {
      setDeployPhase("frontend");
      addLog(`Phase 3: Deploying ${frontends.length} frontend service(s)...`);
      for (const svc of frontends) {
        await deploySvc(svc);
      }
    }

    if (allOk) {
      setDeployPhase("done");
      setDeployDone(true);
      addLog("All services deployed successfully!");
      toast("success", "All services deployed successfully!");
    } else {
      addLog("Deployment completed with errors.");
      toast("error", "Some services failed to deploy.");
    }
    setDeploying(false);
  }, [parseResult, projectId, name, api, toast, exposureOverrides, provisionedServices, addLog, waitForApp]);

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
                <div className="flex items-center gap-3">
                  <button
                    onClick={handleFormatYaml}
                    className="flex items-center gap-1.5 rounded-md border border-border bg-surface-200 px-2.5 py-1 text-xs font-medium text-neutral-300 hover:bg-surface-100 hover:text-white transition-colors"
                    title="Format YAML"
                  >
                    <AlignLeft className="h-3.5 w-3.5" />
                    Format
                  </button>
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
              </div>
              <div className={`flex rounded-lg border bg-surface-200 ${parseErrors.length > 0 ? "border-red-500/50" : "border-border"} focus-within:border-brand`}>
                {/* Line numbers */}
                <div
                  ref={lineNumbersRef}
                  className="select-none overflow-hidden border-r border-border/50 py-3 text-right font-mono text-[11px] leading-[18px] text-neutral-600"
                  style={{ minWidth: "3rem" }}
                >
                  {Array.from({ length: lineCount }, (_, i) => (
                    <div key={i} className="px-2">{i + 1}</div>
                  ))}
                </div>
                <textarea
                  ref={textareaRef}
                  value={composeContent}
                  onChange={(e) => { setComposeContent(e.target.value); setParseErrors([]); setAISuggestions([]); }}
                  onScroll={handleTextareaScroll}
                  placeholder={`version: "3.8"\nservices:\n  api:\n    build: ./api\n    ports:\n      - "8080:8080"\n  db:\n    image: postgres:16`}
                  rows={14}
                  spellCheck={false}
                  className="flex-1 resize-none bg-transparent px-3 py-3 font-mono text-xs leading-[18px] text-white placeholder:text-neutral-600 focus:outline-none"
                />
              </div>
            </div>

            {/* Parse errors (inline) */}
            {parseErrors.length > 0 && (
              <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-4">
                <div className="flex items-center gap-2 text-sm font-medium text-red-400">
                  <XCircle className="h-4 w-4" />
                  Compose Errors
                </div>
                <ul className="mt-2 space-y-1 text-xs text-red-300/80">
                  {parseErrors.map((err, i) => (
                    <li key={i}>• {err}</li>
                  ))}
                </ul>
              </div>
            )}

            {/* AI suggestions */}
            {aiSuggestions.length > 0 && (
              <div className="rounded-lg border border-accent-500/30 bg-accent-500/10 p-4">
                <div className="flex items-center gap-2 text-sm font-medium text-accent-400">
                  <Sparkles className="h-4 w-4" />
                  AI Suggestions
                </div>
                <ul className="mt-2 space-y-1 text-xs text-accent-300/80">
                  {aiSuggestions.map((s, i) => (
                    <li key={i}>• {s}</li>
                  ))}
                </ul>
              </div>
            )}

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
            {(parseResult.warnings || []).length > 0 && (
              <div className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 p-4">
                <div className="flex items-center gap-2 text-sm font-medium text-yellow-400">
                  <AlertTriangle className="h-4 w-4" />
                  Warnings
                </div>
                <ul className="mt-2 space-y-1 text-xs text-yellow-300/80">
                  {(parseResult.warnings || []).map((w, i) => (
                    <li key={i}>• {w}</li>
                  ))}
                </ul>
              </div>
            )}

            {/* App services */}
            <div>
              <h3 className="mb-3 flex items-center gap-2 text-sm font-medium text-neutral-300">
                <Server className="h-4 w-4" />
                App Services ({(parseResult.services || []).length})
              </h3>
              <div className="space-y-3">
                {(parseResult.services || []).map((svc) => (
                  <ServiceCard
                    key={svc.name}
                    service={svc}
                    projectId={projectId}
                    isPublic={exposureOverrides[svc.name] ?? svc.is_public}
                    onToggleExposure={() =>
                      setExposureOverrides((prev) => ({
                        ...prev,
                        [svc.name]: !(prev[svc.name] ?? svc.is_public),
                      }))
                    }
                  />
                ))}
              </div>
            </div>

            {/* Managed services */}
            {(parseResult.managed_services || []).length > 0 && (
              <div>
                <h3 className="mb-3 flex items-center gap-2 text-sm font-medium text-neutral-300">
                  <Database className="h-4 w-4" />
                  Managed Services ({(parseResult.managed_services || []).length})
                </h3>
                <div className="space-y-3">
                  {(parseResult.managed_services || []).map((ms) => (
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
            {(parseResult.services || []).some((s) => s.env_vars.length > 0) && (
              <div>
                <h3 className="mb-3 text-sm font-medium text-neutral-300">
                  Environment Variables
                </h3>
                <div className="rounded-lg border border-border bg-surface-200 p-4">
                  <div className="space-y-2">
                    {(parseResult.services || []).flatMap((svc) =>
                      svc.env_vars.map((ev) => (
                          <div key={`${svc.name}-${ev.key}`} className="text-xs">
                            <span className="text-neutral-400">{svc.name}/</span>
                            <span className="font-medium text-white">{ev.key}</span>
                            {ev.original !== ev.zenith ? (
                              <div className="mt-0.5 flex items-center gap-2 text-neutral-500">
                                <span className="line-through">{ev.original}</span>
                                <ArrowRight className="h-3 w-3" />
                                <span className="text-emerald-400">{ev.zenith}</span>
                              </div>
                            ) : (
                              <div className="mt-0.5 text-neutral-500">{ev.zenith}</div>
                            )}
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
        {step === 3 && parseResult && (
          <div className="space-y-6">
            <div>
              <h2 className="text-base font-medium text-white">Step 3: Deploy</h2>
              <p className="mt-1 text-sm text-neutral-400">
                {deployDone
                  ? "All services are running!"
                  : deploying
                  ? "Deploying your services..."
                  : "Click deploy to launch all services in order: Database → Backend → Frontend"}
              </p>
            </div>

            {/* Phase progress bar */}
            {deploying && (
              <div className="flex items-center gap-2">
                {[
                  { key: "managed", label: "Database", icon: <Database className="h-3 w-3" /> },
                  { key: "backend", label: "Backend", icon: <Lock className="h-3 w-3" /> },
                  { key: "frontend", label: "Frontend", icon: <Globe className="h-3 w-3" /> },
                ].map((phase, i) => {
                  const phases = ["managed", "backend", "frontend", "done"];
                  const currentIdx = phases.indexOf(deployPhase);
                  const phaseIdx = phases.indexOf(phase.key);
                  const isDone = currentIdx > phaseIdx;
                  const isActive = currentIdx === phaseIdx;
                  return (
                    <div key={phase.key} className="flex items-center gap-2 flex-1">
                      <div className={`flex items-center gap-1.5 rounded-full px-3 py-1.5 text-[10px] font-medium transition-all ${
                        isDone ? "bg-emerald-500/20 text-emerald-400" :
                        isActive ? "bg-brand/20 text-brand" :
                        "bg-surface-200 text-neutral-500"
                      }`}>
                        {isDone ? <Check className="h-3 w-3" /> : isActive ? <Loader2 className="h-3 w-3 animate-spin" /> : phase.icon}
                        {phase.label}
                      </div>
                      {i < 2 && <div className={`flex-1 h-px ${isDone ? "bg-emerald-500/40" : "bg-neutral-700"}`} />}
                    </div>
                  );
                })}
              </div>
            )}

            {/* Service cards */}
            <div className="space-y-2">
              {/* Managed services */}
              {(parseResult.managed_services || []).filter((ms) => ms.type === "postgresql").length > 0 && (
                <>
                  <div className="text-[10px] font-medium text-neutral-500 uppercase tracking-wider pt-2">Database</div>
                  {(parseResult.managed_services || []).filter((ms) => ms.type === "postgresql").map((ms) => {
                    const st = deployStatus[`db:${ms.name}`];
                    return (
                      <DeployItemCard
                        key={ms.name}
                        icon={<Database className="h-4 w-4 text-blue-400" />}
                        name={ms.name}
                        meta={`${ms.type} ${ms.version}`}
                        state={st?.state || "pending"}
                        error={st?.error}
                      />
                    );
                  })}
                </>
              )}

              {/* Backend services */}
              {(parseResult.services || []).filter((s) => !(exposureOverrides[s.name] ?? s.is_public)).length > 0 && (
                <>
                  <div className="text-[10px] font-medium text-neutral-500 uppercase tracking-wider pt-3">Backend (Private)</div>
                  {(parseResult.services || []).filter((s) => !(exposureOverrides[s.name] ?? s.is_public)).map((svc) => {
                    const st = deployStatus[svc.name];
                    return (
                      <DeployItemCard
                        key={svc.name}
                        icon={<Lock className="h-4 w-4 text-amber-400" />}
                        name={svc.name}
                        meta={svc.port > 0 ? `:${svc.port}` : undefined}
                        state={st?.state || "pending"}
                        url={st?.url}
                        error={st?.error}
                      />
                    );
                  })}
                </>
              )}

              {/* Frontend services */}
              {(parseResult.services || []).filter((s) => exposureOverrides[s.name] ?? s.is_public).length > 0 && (
                <>
                  <div className="text-[10px] font-medium text-neutral-500 uppercase tracking-wider pt-3">Frontend (Public)</div>
                  {(parseResult.services || []).filter((s) => exposureOverrides[s.name] ?? s.is_public).map((svc) => {
                    const st = deployStatus[svc.name];
                    return (
                      <DeployItemCard
                        key={svc.name}
                        icon={<Globe className="h-4 w-4 text-emerald-400" />}
                        name={svc.name}
                        meta={svc.port > 0 ? `:${svc.port}` : undefined}
                        state={st?.state || "pending"}
                        url={st?.url}
                        error={st?.error}
                      />
                    );
                  })}
                </>
              )}
            </div>

            {/* Deploy log */}
            {deployLog.length > 0 && (
              <div className="rounded-lg border border-border bg-neutral-950 p-4 max-h-48 overflow-y-auto">
                <div className="text-[10px] font-medium text-neutral-500 uppercase tracking-wider mb-2">Deploy Log</div>
                <div className="space-y-0.5 font-mono text-[11px]">
                  {deployLog.map((line, i) => (
                    <div key={i} className={
                      line.includes("✓") ? "text-emerald-400" :
                      line.includes("✗") ? "text-red-400" :
                      line.includes("Phase") ? "text-brand font-semibold" :
                      "text-neutral-400"
                    }>
                      {line}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Success banner with real URLs */}
            {deployDone && (
              <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-4">
                <p className="text-sm font-medium text-emerald-400">
                  Your project is live!
                </p>
                <div className="mt-2 space-y-1">
                  {Object.entries(deployStatus)
                    .filter(([, st]) => st.url)
                    .map(([name, st]) => (
                      <a
                        key={name}
                        href={st.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-2 text-xs text-emerald-300 underline hover:text-emerald-200"
                      >
                        <Globe className="h-3 w-3" />
                        {name}: {st.url}
                      </a>
                    ))}
                </div>
              </div>
            )}

            <div className="flex justify-between">
              {!deployDone && !deploying && (
                <button
                  onClick={() => setStep(2)}
                  className="flex items-center gap-2 rounded-lg border border-border px-4 py-2.5 text-sm text-neutral-300 hover:text-white"
                >
                  <ArrowLeft className="h-4 w-4" />
                  Back
                </button>
              )}
              {deploying && <div />}
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
  isPublic,
  onToggleExposure,
}: {
  service: ParsedService;
  projectId: string;
  isPublic: boolean;
  onToggleExposure: () => void;
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
        </div>
        {service.url && (
          <span className="text-xs text-neutral-400">{service.url}</span>
        )}
      </div>

      {/* Exposure toggle */}
      <div className="mt-3 flex items-center justify-between rounded-lg border border-border/50 bg-neutral-900/50 px-3 py-2">
        <div className="flex items-center gap-2">
          {isPublic ? (
            <>
              <Globe className="h-3.5 w-3.5 text-emerald-400" />
              <span className="text-xs font-medium text-emerald-400">Public</span>
              <span className="text-[10px] text-neutral-500">— accessible via URL</span>
            </>
          ) : (
            <>
              <Lock className="h-3.5 w-3.5 text-amber-400" />
              <span className="text-xs font-medium text-amber-400">Private</span>
              <span className="text-[10px] text-neutral-500">— internal only</span>
            </>
          )}
        </div>
        <button
          onClick={onToggleExposure}
          className={`flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-[10px] font-medium transition-colors ${
            isPublic
              ? "border-amber-500/30 text-amber-400 hover:bg-amber-500/10"
              : "border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/10"
          }`}
        >
          {isPublic ? (
            <>
              <Lock className="h-3 w-3" />
              Make Private
            </>
          ) : (
            <>
              <Globe className="h-3 w-3" />
              Make Public
            </>
          )}
        </button>
      </div>

      {/* API Gateway note for public services */}
      {isPublic && !service.is_public && (
        <div className="mt-2 flex items-center gap-2 rounded-md bg-blue-500/10 px-3 py-1.5">
          <Shield className="h-3 w-3 text-blue-400 shrink-0" />
          <span className="text-[10px] text-blue-300">
            Routed through API Gateway with rate limiting & security
          </span>
        </div>
      )}

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
        {managed.type !== "postgresql" ? (
          <span className="text-xs text-yellow-400">Not supported yet</span>
        ) : provisioned ? (
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

// Deploy item card for Step 3
function DeployItemCard({
  icon,
  name,
  meta,
  state,
  url,
  error,
}: {
  icon: React.ReactNode;
  name: string;
  meta?: string;
  state: "pending" | "creating" | "waiting" | "running" | "error";
  url?: string;
  error?: string;
}) {
  return (
    <div className={`rounded-lg border px-4 py-3 transition-all ${
      state === "running" ? "border-emerald-500/30 bg-emerald-500/5" :
      state === "error" ? "border-red-500/30 bg-red-500/5" :
      state === "creating" || state === "waiting" ? "border-brand/30 bg-brand/5" :
      "border-border bg-surface-200"
    }`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {icon}
          <span className="text-sm font-medium text-white">{name}</span>
          {meta && <span className="text-xs text-neutral-500">{meta}</span>}
        </div>
        <div>
          {state === "running" ? (
            <span className="flex items-center gap-1.5 text-xs text-emerald-400">
              <Check className="h-3.5 w-3.5" />
              Running
            </span>
          ) : state === "creating" ? (
            <span className="flex items-center gap-1.5 text-xs text-brand">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Creating...
            </span>
          ) : state === "waiting" ? (
            <span className="flex items-center gap-1.5 text-xs text-brand">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Starting...
            </span>
          ) : state === "error" ? (
            <span className="flex items-center gap-1.5 text-xs text-red-400">
              <AlertTriangle className="h-3.5 w-3.5" />
              Failed
            </span>
          ) : (
            <span className="text-xs text-neutral-500">Pending</span>
          )}
        </div>
      </div>
      {url && state === "running" && (
        <a href={url} target="_blank" rel="noopener noreferrer" className="mt-2 flex items-center gap-1.5 text-xs text-emerald-300 hover:text-emerald-200 underline">
          <Globe className="h-3 w-3" />
          {url}
        </a>
      )}
      {error && state === "error" && (
        <div className="mt-2 rounded bg-red-500/10 px-3 py-2 text-xs text-red-300 font-mono">
          {error}
        </div>
      )}
    </div>
  );
}

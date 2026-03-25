"use client";

import { Shell } from "@/components/shell";
import { useToast } from "@/components/toast";
import { getApi } from "@/lib/get-api";
import type {
  ComposeImportResult,
  ManagedService,
  ParsedService,
  ParsedManaged,
  RegistryCredentials,
} from "@/lib/api";
import { useRouter } from "next/navigation";
import { useState, useCallback, useRef } from "react";
import yaml from "js-yaml";
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
  Eye,
  EyeOff,
  ExternalLink,
} from "lucide-react";

type Step = 1 | 2 | 3 | 4;

type EnvVarEdit = {
  key: string;
  value: string;
  is_secret: boolean;
  fromCompose: boolean; // pre-filled from compose vs. user-added
};

function looksLikeSecret(key: string): boolean {
  const lower = key.toLowerCase();
  return ["password", "passwd", "secret", "token", "api_key", "apikey",
    "private_key", "privatekey", "auth", "credential", "cert", "key"].some((h) =>
    lower.includes(h)
  );
}

const MANAGED_DEFAULTS: Record<string, Array<{ key: string; value: string }>> = {
  postgresql: [
    { key: "POSTGRES_DB", value: "app" },
    { key: "POSTGRES_USER", value: "app" },
    { key: "POSTGRES_PASSWORD", value: "" },
  ],
  redis: [
    { key: "REDIS_PASSWORD", value: "" },
  ],
  mysql: [
    { key: "MYSQL_DATABASE", value: "app" },
    { key: "MYSQL_USER", value: "app" },
    { key: "MYSQL_ROOT_PASSWORD", value: "" },
    { key: "MYSQL_PASSWORD", value: "" },
  ],
  mongodb: [
    { key: "MONGO_INITDB_ROOT_USERNAME", value: "admin" },
    { key: "MONGO_INITDB_ROOT_PASSWORD", value: "" },
    { key: "MONGO_INITDB_DATABASE", value: "app" },
  ],
  rabbitmq: [
    { key: "RABBITMQ_DEFAULT_USER", value: "app" },
    { key: "RABBITMQ_DEFAULT_PASS", value: "" },
  ],
};

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

  // Step 3: Env vars
  const [envVarEdits, setEnvVarEdits] = useState<Record<string, EnvVarEdit[]>>({});
  // Per-service .env import panel state
  const [importOpenFor, setImportOpenFor] = useState<string | null>(null);
  const [importContent, setImportContent] = useState<Record<string, string>>({});

  // Step 4: Deploy
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

  // Image URL overrides: for build-only services that need a Docker Hub image
  const [imageOverrides, setImageOverrides] = useState<Record<string, string>>({});

  // Confirmed pushes: build-context services where user confirmed they've pushed
  const [confirmedPushes, setConfirmedPushes] = useState<Set<string>>(new Set());

  // Private registry credentials per service (for services using external private registries)
  const [privateRegCreds, setPrivateRegCreds] = useState<Record<string, { username: string; password: string }>>({});

  // Zenith registry credentials (shared robot account — fetched once project is created)
  const [registryCreds, setRegistryCreds] = useState<RegistryCredentials | null>(null);

  // Image verification state (Step 2 → Step 3 gate)
  const [verifying, setVerifying] = useState(false);
  const [verifyResults, setVerifyResults] = useState<Record<string, { reachable: boolean; error?: string }>>({});

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
        // Fetch Zenith registry credentials for Step 2 push instructions
        api.registryCredentials.get(pid).then(setRegistryCreds).catch(() => {});
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

  // Init env var edit state from parse result when entering step 3
  const initEnvVarEdits = useCallback((result: ComposeImportResult) => {
    const edits: Record<string, EnvVarEdit[]> = {};
    // App services — pre-fill from compose
    for (const svc of result.services || []) {
      edits[svc.name] = svc.env_vars.map((ev) => ({
        key: ev.key,
        value: ev.zenith || ev.original || "",
        is_secret: looksLikeSecret(ev.key),
        fromCompose: true,
      }));
      edits[svc.name].push({ key: "", value: "", is_secret: false, fromCompose: false });
    }
    // Managed services — pre-fill with defaults for their type
    for (const ms of result.managed_services || []) {
      const defaults = MANAGED_DEFAULTS[ms.type] || [];
      edits[`_db_${ms.name}`] = defaults.map((d) => ({
        key: d.key,
        value: d.value,
        is_secret: looksLikeSecret(d.key),
        fromCompose: true,
      }));
      edits[`_db_${ms.name}`].push({ key: "", value: "", is_secret: false, fromCompose: false });
    }
    setEnvVarEdits(edits);
  }, []);

  const generatePassword = (): string => {
    const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#%^&*";
    const arr = new Uint8Array(24);
    crypto.getRandomValues(arr);
    return Array.from(arr).map((b) => chars[b % chars.length]).join("");
  };

  // Validate step 3: block if any required env var is empty
  const validateEnvVars = (): string[] => {
    const missing: string[] = [];
    for (const [svcKey, rows] of Object.entries(envVarEdits)) {
      for (const row of rows) {
        if (row.fromCompose && row.key && !row.value) {
          const label = svcKey.startsWith("_db_") ? svcKey.slice(4) : svcKey;
          missing.push(`${label} / ${row.key}`);
        }
      }
    }
    return missing;
  };

  const handleEnvVarChange = (svcName: string, idx: number, field: keyof EnvVarEdit, val: string | boolean) => {
    setEnvVarEdits((prev) => {
      const rows = [...(prev[svcName] || [])];
      rows[idx] = { ...rows[idx], [field]: val };
      return { ...prev, [svcName]: rows };
    });
  };

  const addEnvVarRow = (svcName: string) => {
    setEnvVarEdits((prev) => ({
      ...prev,
      [svcName]: [...(prev[svcName] || []), { key: "", value: "", is_secret: false, fromCompose: false }],
    }));
  };

  const removeEnvVarRow = (svcName: string, idx: number) => {
    setEnvVarEdits((prev) => {
      const rows = (prev[svcName] || []).filter((_, i) => i !== idx);
      return { ...prev, [svcName]: rows };
    });
  };

  // Parse and merge a .env file content into a specific service's env vars
  const applyDotEnvImport = (svcName: string, content: string) => {
    const parsed: Record<string, string> = {};
    content.split("\n").forEach((line) => {
      line = line.trim();
      if (!line || line.startsWith("#")) return;
      line = line.replace(/^export\s+/, "");
      const eqIdx = line.indexOf("=");
      if (eqIdx < 1) return;
      const key = line.slice(0, eqIdx).trim();
      let value = line.slice(eqIdx + 1).trim();
      if (value.length >= 2 && ((value[0] === '"' && value.endsWith('"')) || (value[0] === "'" && value.endsWith("'")))) {
        value = value.slice(1, -1);
      }
      if (/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) parsed[key] = value;
    });
    if (Object.keys(parsed).length === 0) return;
    setEnvVarEdits((prev) => {
      const rows = [...(prev[svcName] || [])];
      for (const [key, value] of Object.entries(parsed)) {
        const existing = rows.findIndex((r) => r.key === key);
        if (existing >= 0) {
          rows[existing] = { ...rows[existing], value };
        } else {
          const insertAt = rows.length > 0 && !rows[rows.length - 1].key ? rows.length - 1 : rows.length;
          rows.splice(insertAt, 0, { key, value, is_secret: looksLikeSecret(key), fromCompose: false });
        }
      }
      return { ...prev, [svcName]: rows };
    });
    setImportContent((prev) => ({ ...prev, [svcName]: "" }));
    setImportOpenFor(null);
  };

  // Format YAML: parse and re-serialize for correct indentation
  // Falls back to AI formatting if js-yaml can't parse
  const handleFormatYaml = useCallback(async () => {
    try {
      const parsed = yaml.load(composeContent);
      if (!parsed || typeof parsed !== "object") {
        throw new Error("Not a valid YAML object");
      }
      const formatted = yaml.dump(parsed, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false,
        quotingType: '"',
        forceQuotes: false,
      });
      setComposeContent(formatted);
      setParseErrors([]);
      setAISuggestions([]);
      toast("success", "YAML formatted");
    } catch {
      // js-yaml failed — try AI-powered format
      if (!projectId) {
        toast("error", "Create a project first to use AI formatting");
        return;
      }
      toast("info", "YAML has errors, using AI to fix...");
      try {
        const result = await api.composeImport.format(projectId, composeContent);
        if (result.formatted) {
          setComposeContent(result.formatted);
          setParseErrors([]);
          setAISuggestions([]);
          toast("success", "YAML fixed by AI");
        } else {
          toast("error", "AI could not fix the YAML");
        }
      } catch {
        toast("error", "AI formatting failed — check your YAML manually");
      }
    }
  }, [composeContent, toast, projectId, api]);

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

  // Poll app until running or error (max 90s)
  const waitForApp = useCallback(async (appId: string, appName: string, imageUrl: string): Promise<{ ok: boolean; url?: string; error?: string }> => {
    const maxAttempts = 30;
    for (let i = 0; i < maxAttempts; i++) {
      await new Promise((r) => setTimeout(r, 3000));
      try {
        const app = await api.appsDeploy.get(appId);
        if (app.status === "running" || app.status === "active") {
          return { ok: true, url: app.url };
        }
        if (app.status === "error" || app.status === "failed" || app.status === "crash_loop") {
          const hint = imageUrl.includes("registry.stage.freezenith.com")
            ? `Image not found in registry. Push your image first:\n  docker build -t ${imageUrl} .\n  docker push ${imageUrl}`
            : app.status === "crash_loop"
            ? "App is crash-looping. Check your env vars and startup command. View the Logs tab for details."
            : "App failed to start. Check your env vars and image. View the Logs tab for details.";
          return { ok: false, error: hint };
        }
        // Still deploying...
      } catch {
        // Ignore transient errors during polling
      }
    }
    return { ok: false, error: "Timed out (90s). The app may still be starting — check the Logs tab." };
  }, [api]);

  // Step 3: Deploy with phases: managed → backend → frontend
  const handleDeploy = useCallback(async () => {
    if (!parseResult || !projectId) return;
    setDeploying(true);
    setDeployLog([]);

    const slug = name.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
    const services = parseResult.services || [];
    const managed = parseResult.managed_services || [];

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

      let imageUrl = imageOverrides[svc.name] || svc.image || "";
      if (!imageUrl && svc.build_context) {
        imageUrl = `registry.stage.freezenith.com/${projectId}/${svc.name}:latest`;
      }

      const envVars = (envVarEdits[svc.name] || svc.env_vars.map((ev) => ({
        key: ev.key, value: ev.zenith || ev.original || "", is_secret: false, fromCompose: true,
      })))
        .filter((ev) => ev.key.trim() && ev.value.trim())
        .map((ev) => ({ key: ev.key.trim(), value: ev.value }));

      const appName = `${slug}-${svc.name}`;
      const isPublic = exposureOverrides[svc.name] ?? svc.is_public;

      if (svc.build_context && !svc.image && !imageOverrides[svc.name]) {
        addLog(`⚠ ${svc.name} uses build: context — push image first:`);
        addLog(`  docker build -t ${imageUrl} .`);
        addLog(`  docker push ${imageUrl}`);
      }
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

        const result = await waitForApp(app.id, appName, imageUrl);
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
  }, [parseResult, projectId, name, api, toast, exposureOverrides, imageOverrides, provisionedServices, addLog, waitForApp, envVarEdits]);

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
            <div className="h-px w-4 bg-neutral-700" />
            <StepDot active={step >= 2} done={step > 2} label="2" />
            <div className="h-px w-4 bg-neutral-700" />
            <StepDot active={step >= 3} done={step > 3} label="3" />
            <div className="h-px w-4 bg-neutral-700" />
            <StepDot active={step >= 4} done={deployDone} label="4" />
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
        {step === 2 && parseResult && (() => {
          const services = parseResult.services || [];
          const buildServices = services.filter((s) => s.build_context && !imageOverrides[s.name]);
          const unresolvedBuilds = buildServices.filter((s) => !confirmedPushes.has(s.name));

          return (
          <div className="space-y-6">
            <div>
              <h2 className="text-base font-medium text-white">Step 2: Review Services</h2>
              <p className="mt-1 text-sm text-neutral-400">
                Zenith detected {services.length} app service{services.length !== 1 ? "s" : ""} and {(parseResult.managed_services || []).length} managed service{(parseResult.managed_services || []).length !== 1 ? "s" : ""}.
              </p>
            </div>

            {/* Action Required — build-context services */}
            {unresolvedBuilds.length > 0 && (
              <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 px-4 py-3 flex items-start gap-3">
                <AlertTriangle className="h-4 w-4 shrink-0 text-amber-400 mt-0.5" />
                <div>
                  <p className="text-sm font-semibold text-amber-400">
                    {unresolvedBuilds.length} service{unresolvedBuilds.length !== 1 ? "s" : ""} need{unresolvedBuilds.length === 1 ? "s" : ""} a Docker image
                  </p>
                  <p className="text-xs text-amber-300/70 mt-0.5">
                    Zenith deploys images, not source code. For each service below, pick how to provide the image.
                  </p>
                </div>
              </div>
            )}

            {/* All resolved */}
            {unresolvedBuilds.length === 0 && buildServices.length > 0 && (
              <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-4 py-3">
                <div className="flex items-center gap-2 text-sm text-emerald-400">
                  <Check className="h-4 w-4" />
                  All services have images — ready to configure env vars.
                </div>
              </div>
            )}

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
                App Services ({services.length})
              </h3>
              <div className="space-y-3">
                {services.map((svc) => (
                  <ServiceCard
                    key={svc.name}
                    service={svc}
                    projectId={projectId}
                    isPublic={exposureOverrides[svc.name] ?? svc.is_public}
                    imageOverride={imageOverrides[svc.name] || ""}
                    confirmed={confirmedPushes.has(svc.name)}
                    registryCreds={registryCreds}
                    privateRegCred={privateRegCreds[svc.name] || { username: "", password: "" }}
                    onPrivateRegCredChange={(cred) =>
                      setPrivateRegCreds((prev) => ({ ...prev, [svc.name]: cred }))
                    }
                    onToggleExposure={() =>
                      setExposureOverrides((prev) => ({
                        ...prev,
                        [svc.name]: !(prev[svc.name] ?? svc.is_public),
                      }))
                    }
                    onImageChange={(url) => {
                      setImageOverrides((prev) => ({ ...prev, [svc.name]: url }));
                      // Clear confirmed if user changes image
                      if (!url) {
                        setConfirmedPushes((prev) => { const s = new Set(prev); s.delete(svc.name); return s; });
                      }
                    }}
                    onConfirm={() =>
                      setConfirmedPushes((prev) => new Set([...prev, svc.name]))
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

            {/* Per-service verification results */}
            {Object.keys(verifyResults).length > 0 && (
              <div className="space-y-1.5">
                {services.map((svc) => {
                  const res = verifyResults[svc.name];
                  if (!res) return null;
                  return (
                    <div
                      key={svc.name}
                      className={`flex items-center gap-2 rounded-lg px-3 py-2 text-xs ${
                        res.reachable
                          ? "bg-emerald-500/10 border border-emerald-500/20 text-emerald-400"
                          : "bg-red-500/10 border border-red-500/20 text-red-400"
                      }`}
                    >
                      {res.reachable ? (
                        <Check className="h-3.5 w-3.5 shrink-0" />
                      ) : (
                        <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
                      )}
                      <span className="font-medium">{svc.name}</span>
                      <span className="text-[10px] opacity-70">
                        {res.reachable ? "image verified ✓" : res.error || "not reachable"}
                      </span>
                    </div>
                  );
                })}
              </div>
            )}

            <div className="flex justify-between">
              <button
                onClick={() => { setVerifyResults({}); setStep(1); }}
                className="flex items-center gap-2 rounded-lg border border-border px-4 py-2.5 text-sm text-neutral-300 hover:text-white"
              >
                <ArrowLeft className="h-4 w-4" />
                Back
              </button>
              <button
                disabled={verifying}
                onClick={async () => {
                  if (unresolvedBuilds.length > 0) {
                    toast("error", `Resolve images for: ${unresolvedBuilds.map((s) => s.name).join(", ")}. Provide an image URL or confirm you've pushed.`);
                    return;
                  }

                  // Build the list of images to verify
                  const imagesToVerify = services.map((svc) => ({
                    name: svc.name,
                    image: imageOverrides[svc.name] || svc.image ||
                      `registry.stage.freezenith.com/${projectId}/${svc.name}:latest`,
                  }));

                  setVerifying(true);
                  setVerifyResults({});
                  try {
                    const result = await api.imageVerify.verify(projectId, imagesToVerify);
                    const resultMap: Record<string, { reachable: boolean; error?: string }> = {};
                    for (const r of result.results) {
                      resultMap[r.name] = { reachable: r.reachable, error: r.error };
                    }
                    setVerifyResults(resultMap);

                    if (!result.all_ready) {
                      const failed = result.results.filter((r) => !r.reachable).map((r) => r.name);
                      toast("error", `Images not ready: ${failed.join(", ")}. Push them and try again.`);
                      return;
                    }
                  } catch {
                    // Verification failed (network error, etc.) — don't block proceed
                    toast("warning", "Could not verify images (registry unreachable). Proceeding anyway.");
                  } finally {
                    setVerifying(false);
                  }

                  if (parseResult) initEnvVarEdits(parseResult);
                  setStep(3);
                }}
                className="flex items-center gap-2 rounded-lg bg-brand px-5 py-2.5 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-60"
              >
                {verifying ? (
                  <><span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />Verifying images...</>
                ) : (
                  <><ArrowRight className="h-4 w-4" />Verify &amp; Configure Env Vars</>
                )}
              </button>
            </div>
          </div>
          );
        })()}

        {/* Step 3: Env Vars */}
        {step === 3 && parseResult && (() => {
          // Build list of all sections: app services + managed services
          const sections: Array<{ key: string; label: string; icon: React.ReactNode; isManaged: boolean }> = [
            ...(parseResult.services || []).map((svc) => ({
              key: svc.name,
              label: svc.name,
              icon: <Server className="h-4 w-4 text-accent-400" />,
              isManaged: false,
            })),
            ...(parseResult.managed_services || []).map((ms) => ({
              key: `_db_${ms.name}`,
              label: ms.name,
              icon: <Database className="h-4 w-4 text-blue-400" />,
              isManaged: true,
            })),
          ];

          return (
            <div className="space-y-6">
              <div>
                <h2 className="text-base font-medium text-white">Step 3: Environment Variables</h2>
                <p className="mt-1 text-sm text-neutral-400">
                  Set environment variables for each service. Amber rows need a value.
                </p>
              </div>

              {sections.map(({ key: svcKey, label, icon, isManaged }) => {
                const rows = envVarEdits[svcKey] || [];
                const emptyRequired = rows.filter((r) => r.fromCompose && r.key && !r.value).length;
                const isImportOpen = importOpenFor === svcKey;

                return (
                  <div key={svcKey} className="space-y-2">
                    {/* Service header */}
                    <div className="flex items-center justify-between">
                      <h3 className="flex items-center gap-2 text-sm font-medium text-neutral-300">
                        {icon}
                        {label}
                        {isManaged && (
                          <span className="rounded-full bg-blue-500/20 px-2 py-0.5 text-[10px] text-blue-400">managed</span>
                        )}
                        {emptyRequired > 0 && (
                          <span className="rounded-full bg-amber-500/20 px-2 py-0.5 text-[10px] text-amber-400">
                            {emptyRequired} empty
                          </span>
                        )}
                      </h3>
                      <button
                        onClick={() => setImportOpenFor(isImportOpen ? null : svcKey)}
                        className="flex items-center gap-1.5 rounded border border-border px-2.5 py-1 text-xs text-neutral-400 hover:text-white transition-colors"
                      >
                        <Upload className="h-3 w-3" />
                        Import .env
                      </button>
                    </div>

                    {/* Inline .env import panel */}
                    {isImportOpen && (
                      <div className="rounded-lg border border-accent-500/30 bg-accent-500/5 p-3 space-y-2">
                        <div className="flex items-center justify-between">
                          <p className="text-xs text-neutral-400">Paste .env content for <span className="text-white font-medium">{label}</span></p>
                          <label className="flex cursor-pointer items-center gap-1 rounded border border-border px-2 py-0.5 text-[10px] text-neutral-500 hover:text-white transition-colors">
                            <Upload className="h-2.5 w-2.5" /> Upload file
                            <input
                              type="file"
                              accept=".env,text/plain"
                              className="hidden"
                              onChange={(e) => {
                                const file = e.target.files?.[0];
                                if (!file) return;
                                const reader = new FileReader();
                                reader.onload = (ev) => {
                                  const content = ev.target?.result as string ?? "";
                                  applyDotEnvImport(svcKey, content);
                                };
                                reader.readAsText(file);
                                e.target.value = "";
                              }}
                            />
                          </label>
                        </div>
                        <textarea
                          value={importContent[svcKey] || ""}
                          onChange={(e) => setImportContent((prev) => ({ ...prev, [svcKey]: e.target.value }))}
                          placeholder={"DATABASE_URL=postgres://...\nJWT_SECRET=..."}
                          rows={4}
                          className="w-full rounded border border-border bg-surface-200 px-3 py-2 font-mono text-xs text-neutral-300 placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none resize-none"
                        />
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => setImportOpenFor(null)}
                            className="rounded px-3 py-1 text-xs text-neutral-500 hover:text-white transition-colors"
                          >
                            Cancel
                          </button>
                          <button
                            onClick={() => applyDotEnvImport(svcKey, importContent[svcKey] || "")}
                            disabled={!(importContent[svcKey] || "").trim()}
                            className="rounded bg-accent-500 px-3 py-1 text-xs font-medium text-white hover:bg-accent-600 disabled:opacity-40 transition-colors"
                          >
                            Apply
                          </button>
                        </div>
                      </div>
                    )}

                    {/* Env var table */}
                    <div className="rounded-lg border border-border bg-surface-200 overflow-hidden">
                      <div className="grid grid-cols-[1fr_1fr_5rem_2rem] gap-2 border-b border-border bg-surface-100 px-3 py-2 text-[10px] font-medium uppercase tracking-wider text-neutral-500">
                        <span>Key</span><span>Value</span><span>Secret</span><span />
                      </div>
                      {rows.map((row, idx) => {
                        const isRequired = row.fromCompose && row.key && !row.value;
                        const canGenerate = isRequired && looksLikeSecret(row.key);
                        return (
                        <div
                          key={idx}
                          className={`grid grid-cols-[1fr_1fr_5rem_2rem] gap-2 items-center px-3 py-1.5 border-b border-border/50 last:border-0 ${
                            isRequired ? "bg-amber-500/5" : ""
                          }`}
                        >
                          <input
                            type="text"
                            value={row.key}
                            onChange={(e) => handleEnvVarChange(svcKey, idx, "key", e.target.value)}
                            placeholder="KEY"
                            className="w-full rounded bg-transparent px-2 py-1 font-mono text-xs text-white placeholder:text-neutral-600 border border-transparent hover:border-border focus:border-accent-500 focus:outline-none"
                          />
                          <div className="flex items-center gap-1 min-w-0">
                            <input
                              type={row.is_secret ? "password" : "text"}
                              value={row.value}
                              onChange={(e) => handleEnvVarChange(svcKey, idx, "value", e.target.value)}
                              placeholder={isRequired ? "required" : "value"}
                              className={`min-w-0 flex-1 rounded bg-transparent px-2 py-1 font-mono text-xs border border-transparent hover:border-border focus:border-accent-500 focus:outline-none ${
                                isRequired
                                  ? "placeholder:text-amber-500/70 text-white"
                                  : "placeholder:text-neutral-600 text-neutral-300"
                              }`}
                            />
                            {canGenerate && (
                              <button
                                onClick={() => handleEnvVarChange(svcKey, idx, "value", generatePassword())}
                                title="Generate strong password"
                                className="shrink-0 rounded bg-accent-500/20 px-1.5 py-0.5 text-[9px] font-medium text-accent-400 hover:bg-accent-500/30 transition-colors whitespace-nowrap"
                              >
                                Generate
                              </button>
                            )}
                          </div>
                          <button
                            onClick={() => handleEnvVarChange(svcKey, idx, "is_secret", !row.is_secret)}
                            className={`flex items-center gap-1 rounded px-2 py-1 text-[10px] transition-colors ${
                              row.is_secret ? "bg-amber-500/20 text-amber-400" : "text-neutral-600 hover:text-neutral-400"
                            }`}
                          >
                            <Lock className="h-3 w-3" />
                            {row.is_secret ? "Secret" : "Plain"}
                          </button>
                          <button
                            onClick={() => removeEnvVarRow(svcKey, idx)}
                            className="text-neutral-600 hover:text-red-400 transition-colors"
                          >
                            <XCircle className="h-3.5 w-3.5" />
                          </button>
                        </div>
                        );
                      })}
                      <button
                        onClick={() => addEnvVarRow(svcKey)}
                        className="flex w-full items-center gap-2 px-3 py-2 text-xs text-neutral-500 hover:text-accent-400 hover:bg-surface-100 transition-colors"
                      >
                        <span className="text-base leading-none">+</span> Add variable
                      </button>
                    </div>
                  </div>
                );
              })}

              <div className="flex justify-between">
                <button
                  onClick={() => setStep(2)}
                  className="flex items-center gap-2 rounded-lg border border-border px-4 py-2.5 text-sm text-neutral-300 hover:text-white"
                >
                  <ArrowLeft className="h-4 w-4" />
                  Back
                </button>
                <button
                  onClick={() => {
                    const missing = validateEnvVars();
                    if (missing.length > 0) {
                      toast("error", `Fill required fields first: ${missing.join(", ")}`);
                      return;
                    }
                    setStep(4);
                  }}
                  className="flex items-center gap-2 rounded-lg bg-brand px-5 py-2.5 text-sm font-medium text-white hover:bg-brand/90"
                >
                  <Rocket className="h-4 w-4" />
                  Deploy
                </button>
              </div>
            </div>
          );
        })()}

        {/* Step 4: Deploy */}
        {step === 4 && parseResult && (
          <div className="space-y-6">
            <div>
              <h2 className="text-base font-medium text-white">Step 4: Deploy</h2>
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
              {(parseResult.managed_services || []).length > 0 && (
                <>
                  <div className="text-[10px] font-medium text-neutral-500 uppercase tracking-wider pt-2">Managed Services</div>
                  {(parseResult.managed_services || []).map((ms) => {
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
                  onClick={() => setStep(3)}
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

function CopyLine({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div className="flex items-center gap-2">
      <span className="w-8 shrink-0 text-[9px] font-medium uppercase tracking-wider text-neutral-600">{label}</span>
      <code className="flex-1 rounded bg-neutral-900 px-2.5 py-1.5 font-mono text-[11px] text-neutral-300 overflow-x-auto whitespace-nowrap">
        {value}
      </code>
      <button
        onClick={handleCopy}
        title="Copy"
        className="shrink-0 rounded border border-border p-1 text-neutral-500 hover:text-white transition-colors"
      >
        {copied ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
      </button>
    </div>
  );
}

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

function isPrivateRegistryHost(url: string): boolean {
  if (!url) return false;
  const firstSlash = url.indexOf("/");
  if (firstSlash === -1) return false;
  const seg = url.substring(0, firstSlash);
  return seg.includes(".") || seg.includes(":");
}

function ServiceCard({
  service,
  projectId,
  isPublic,
  imageOverride,
  confirmed,
  registryCreds,
  privateRegCred,
  onPrivateRegCredChange,
  onToggleExposure,
  onImageChange,
  onConfirm,
}: {
  service: ParsedService;
  projectId: string;
  isPublic: boolean;
  imageOverride: string;
  confirmed: boolean;
  registryCreds: import("@/lib/api").RegistryCredentials | null;
  privateRegCred: { username: string; password: string };
  onPrivateRegCredChange: (cred: { username: string; password: string }) => void;
  onToggleExposure: () => void;
  onImageChange: (url: string) => void;
  onConfirm: () => void;
}) {
  const [showRegPass, setShowRegPass] = useState(false);
  const hasBuildContext = !!service.build_context;
  const isResolved = !hasBuildContext || !!imageOverride || confirmed;
  const zenithPushTarget = `${registryCreds?.push_prefix ?? `registry.stage.freezenith.com/${projectId}`}/${service.name}:latest`;
  const isPrivate = isPrivateRegistryHost(imageOverride);

  return (
    <div className={`rounded-lg border bg-surface-200 p-4 transition-colors ${
      hasBuildContext && !isResolved ? "border-amber-500/40" : isResolved ? "border-emerald-500/20" : "border-border"
    }`}>
      {/* Header */}
      <div className="flex items-center gap-3">
        <Server className={`h-4 w-4 shrink-0 ${hasBuildContext && !isResolved ? "text-amber-400" : isResolved ? "text-emerald-400" : "text-brand"}`} />
        <span className="text-sm font-medium text-white">{service.name}</span>
        {service.port > 0 && (
          <span className="rounded bg-neutral-700 px-1.5 py-0.5 text-[10px] text-neutral-300">:{service.port}</span>
        )}
        {hasBuildContext && !isResolved && (
          <span className="rounded-full bg-amber-500/20 px-2 py-0.5 text-[10px] font-medium text-amber-400">no image yet</span>
        )}
        {isResolved && hasBuildContext && (
          <span className="flex items-center gap-1 rounded-full bg-emerald-500/20 px-2 py-0.5 text-[10px] font-medium text-emerald-400">
            <Check className="h-2.5 w-2.5" /> image ready
          </span>
        )}
        {!hasBuildContext && service.image && (
          <span className="rounded-full bg-emerald-500/20 px-2 py-0.5 text-[10px] font-medium text-emerald-400">image ready</span>
        )}
      </div>

      {/* Public image — nothing to do */}
      {!hasBuildContext && service.image && (
        <div className="mt-2 flex items-center gap-2 rounded-md bg-emerald-500/5 border border-emerald-500/20 px-3 py-2">
          <Check className="h-3 w-3 text-emerald-400 shrink-0" />
          <span className="font-mono text-[11px] text-neutral-300">{service.image}</span>
          <span className="ml-auto text-[10px] text-neutral-500">pulled automatically at deploy</span>
        </div>
      )}

      {/* Build context — needs image resolution */}
      {hasBuildContext && (
        <div className="mt-3 space-y-3">
          <p className="text-[11px] text-neutral-500">
            Uses <code className="bg-neutral-800 px-1 rounded text-neutral-300">build: {service.build_context}</code> — Zenith deploys images, not source code. Choose one option below:
          </p>

          {/* ── Path 1: Already have an image ── */}
          <div className="rounded-lg border border-border bg-neutral-900/60 p-3 space-y-2">
            <p className="text-[11px] font-semibold text-white">
              I already have a Docker image
            </p>
            <p className="text-[10px] text-neutral-500">
              Enter the full image URL — Docker Hub, GHCR, or any registry you use.
            </p>
            <input
              type="text"
              value={imageOverride}
              onChange={(e) => onImageChange(e.target.value)}
              placeholder="e.g. myuser/backend:latest  or  ghcr.io/org/backend:v1.0"
              className={`w-full rounded bg-neutral-800 px-3 py-2 font-mono text-[11px] placeholder:text-neutral-600 border focus:outline-none ${
                imageOverride ? "border-emerald-500/40 text-emerald-300 focus:border-emerald-500" : "border-border text-neutral-300 focus:border-accent-500"
              }`}
            />
            {/* Private registry credential fields */}
            {isPrivate && (
              <div className="space-y-1.5 pt-1">
                <p className="text-[10px] text-amber-400/80">Private registry detected — enter credentials so Zenith can pull the image:</p>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={privateRegCred.username}
                    onChange={(e) => onPrivateRegCredChange({ ...privateRegCred, username: e.target.value })}
                    placeholder="Username"
                    className="flex-1 rounded bg-neutral-800 border border-border px-2 py-1.5 font-mono text-[11px] text-neutral-300 placeholder:text-neutral-600 focus:outline-none focus:border-accent-500"
                  />
                  <input
                    type="password"
                    value={privateRegCred.password}
                    onChange={(e) => onPrivateRegCredChange({ ...privateRegCred, password: e.target.value })}
                    placeholder="Password / token"
                    className="flex-1 rounded bg-neutral-800 border border-border px-2 py-1.5 font-mono text-[11px] text-neutral-300 placeholder:text-neutral-600 focus:outline-none focus:border-accent-500"
                  />
                </div>
              </div>
            )}
          </div>

          {/* ── Divider ── */}
          <div className="flex items-center gap-2 text-[10px] text-neutral-600">
            <div className="flex-1 h-px bg-neutral-800" />
            or
            <div className="flex-1 h-px bg-neutral-800" />
          </div>

          {/* ── Path 2: Push to Zenith registry ── */}
          <div className="rounded-lg border border-border bg-neutral-900/60 p-3 space-y-2">
            <p className="text-[11px] font-semibold text-white">
              Build and push to Zenith registry
            </p>
            <p className="text-[10px] text-neutral-500">
              Use the private registry included with your project. Run these commands once:
            </p>

            {/* Login */}
            <div className="space-y-1">
              <p className="text-[10px] text-neutral-500 uppercase tracking-wider font-medium">1 — Login</p>
              {registryCreds?.available ? (
                <div className="rounded bg-neutral-800 border border-neutral-700 px-3 py-2 space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] text-neutral-500 w-14 shrink-0">user</span>
                    <code className="flex-1 font-mono text-[11px] text-neutral-300">{registryCreds.username}</code>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] text-neutral-500 w-14 shrink-0">password</span>
                    <code className="flex-1 font-mono text-[11px] text-neutral-300">
                      {showRegPass ? registryCreds.password : "••••••••••••"}
                    </code>
                    <button onClick={() => setShowRegPass((v) => !v)} className="shrink-0 text-neutral-500 hover:text-white">
                      {showRegPass ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
                    </button>
                  </div>
                </div>
              ) : (
                <div className="rounded bg-neutral-800 border border-neutral-700 px-3 py-2">
                  <code className="font-mono text-[11px] text-neutral-500">Loading credentials...</code>
                </div>
              )}
              <CopyLine
                label="login"
                value={`docker login ${registryCreds?.push_prefix?.split("/")[0] ?? "registry.stage.freezenith.com"} -u '${registryCreds?.username ?? "<user>"}' -p '${registryCreds?.password ?? "<pass>"}'`}
              />
            </div>

            {/* Build + Push */}
            <div className="space-y-1">
              <p className="text-[10px] text-neutral-500 uppercase tracking-wider font-medium">2 — Build &amp; Push</p>
              <CopyLine label="build" value={`docker build -t ${zenithPushTarget} ${service.build_context || "."}`} />
              <CopyLine label="push" value={`docker push ${zenithPushTarget}`} />
              <p className="text-[10px] text-neutral-600">
                The <code className="text-neutral-500">-t</code> flag names the image with the registry address — this is what tells Docker where to push it.
              </p>
            </div>

            {/* Confirm button */}
            {!imageOverride && (
              <button
                onClick={onConfirm}
                disabled={confirmed}
                className={`mt-1 flex w-full items-center justify-center gap-2 rounded-lg border py-2 text-xs font-medium transition-colors ${
                  confirmed
                    ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-400 cursor-default"
                    : "border-brand/40 text-brand hover:bg-brand/10"
                }`}
              >
                {confirmed ? (
                  <><Check className="h-3.5 w-3.5" /> Image pushed — ready to deploy</>
                ) : (
                  <><Check className="h-3.5 w-3.5" /> I&apos;ve pushed this image</>
                )}
              </button>
            )}
          </div>

          {/* ── Tip: GitHub Actions ── */}
          <div className="flex items-start gap-2 rounded-lg border border-blue-500/20 bg-blue-500/5 px-3 py-2">
            <span className="text-blue-400 shrink-0 mt-0.5">💡</span>
            <p className="text-[11px] text-blue-300/80">
              <span className="font-medium text-blue-300">Automate this.</span>{" "}
              Set up GitHub Actions to build and push images automatically on every commit —{" "}
              <a href="/ci" className="underline hover:text-blue-200 inline-flex items-center gap-0.5">
                CI/CD settings <ExternalLink className="h-2.5 w-2.5" />
              </a>
            </p>
          </div>
        </div>
      )}

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
            <><Lock className="h-3 w-3" /> Make Private</>
          ) : (
            <><Globe className="h-3 w-3" /> Make Public</>
          )}
        </button>
      </div>

      {/* API Gateway note */}
      {isPublic && !service.is_public && (
        <div className="mt-2 flex items-center gap-2 rounded-md bg-blue-500/10 px-3 py-1.5">
          <Shield className="h-3 w-3 text-blue-400 shrink-0" />
          <span className="text-[10px] text-blue-300">
            Routed through API Gateway with rate limiting &amp; security
          </span>
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
  const typeConfig: Record<string, { icon: string; color: string }> = {
    postgresql: { icon: "P", color: "bg-blue-500/20 text-blue-400" },
    redis: { icon: "R", color: "bg-red-500/20 text-red-400" },
    mysql: { icon: "M", color: "bg-orange-500/20 text-orange-400" },
    mongodb: { icon: "M", color: "bg-green-500/20 text-green-400" },
    rabbitmq: { icon: "Q", color: "bg-purple-500/20 text-purple-400" },
  };
  const cfg = typeConfig[managed.type] || { icon: "?", color: "bg-neutral-500/20 text-neutral-400" };

  return (
    <div className="flex items-center justify-between rounded-lg border border-border bg-surface-200 px-4 py-3">
      <div className="flex items-center gap-3">
        <span
          className={`flex h-6 w-6 items-center justify-center rounded text-[10px] font-bold ${cfg.color}`}
        >
          {cfg.icon}
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

"use client";

import { useState, useEffect } from "react";
import { Modal } from "@/components/modal";
import { useToast } from "@/components/toast";
import { getApi } from "@/lib/get-api";
import type { RegistryImage, StorageBucket, Database as DbType, AppType, AppExposure } from "@/lib/api";
import {
  Rocket,
  Container,
  Lock,
  Crown,
  Eye,
  EyeOff,
  Search,
  Plus,
  Trash2,
  Check,
  Database,
  Loader2,
  ChevronDown,
  Tag,
  HardDrive,
  Cpu,
  Globe,
  Cog,
  Clock,
  Shield,
} from "lucide-react";

// ── Well-known image catalog ──

const WELL_KNOWN_IMAGES: Record<string, { port: number; description: string }> = {
  "nginx":            { port: 80,    description: "Web server" },
  "httpd":            { port: 80,    description: "Apache HTTP" },
  "node":             { port: 3000,  description: "Node.js" },
  "python":           { port: 8000,  description: "Python" },
  "golang":           { port: 8080,  description: "Go" },
  "redis":            { port: 6379,  description: "Redis" },
  "postgres":         { port: 5432,  description: "PostgreSQL" },
  "mysql":            { port: 3306,  description: "MySQL" },
  "mongo":            { port: 27017, description: "MongoDB" },
  "traefik":          { port: 80,    description: "Traefik proxy" },
  "caddy":            { port: 80,    description: "Caddy server" },
  "grafana/grafana":  { port: 3000,  description: "Grafana" },
  "prom/prometheus":  { port: 9090,  description: "Prometheus" },
};

/** Extract base image name from a ref (strip registry + tag). */
function extractBaseImage(ref: string): string {
  ref = ref.trim();
  if (!ref) return "";
  const parts = ref.split("/");
  let name: string;
  if (parts.length >= 2 && (parts[0].includes(".") || parts[0].includes(":"))) {
    name = parts.slice(1).join("/");
  } else {
    name = ref;
  }
  // Strip "library/" prefix
  if (name.startsWith("library/")) name = name.slice(8);
  // Strip tag
  const colonIdx = name.lastIndexOf(":");
  if (colonIdx > 0) name = name.slice(0, colonIdx);
  return name;
}

/** Detect well-known image and return its info, or null. */
function detectWellKnownImage(ref: string): { name: string; port: number; description: string } | null {
  const base = extractBaseImage(ref);
  if (!base) return null;
  const match = WELL_KNOWN_IMAGES[base];
  if (match) return { name: base, ...match };
  return null;
}

// ── Image ref normalizer (mirrors API-side logic) ──

function normalizeImageRef(ref: string): string {
  ref = ref.trim();
  if (!ref) return ref;
  const addTag = !ref.includes(":");
  const parts = ref.split("/");
  const hasRegistry = parts.length > 1 && (parts[0].includes(".") || parts[0].includes(":"));
  if (!hasRegistry) {
    ref = ref.includes("/") ? "docker.io/" + ref : "docker.io/library/" + ref;
  }
  if (addTag) ref += ":latest";
  return ref;
}

// ── Types ──

interface EnvEntry {
  key: string;
  value: string;
}

// Plan-tier resource limits (read-only display)
const PLAN_RESOURCES: Record<string, { cpu: string; ram: string }> = {
  free:       { cpu: "0.25 vCPU", ram: "256 MB" },
  pro:        { cpu: "0.5 vCPU",  ram: "512 MB" },
  team:       { cpu: "1 vCPU",    ram: "1 GB" },
  business:   { cpu: "2 vCPU",    ram: "2 GB" },
  enterprise: { cpu: "4 vCPU",    ram: "4 GB" },
};

// User-facing category → maps to appType + exposure
type AppCategory = "frontend" | "api" | "background";

interface WizardState {
  // Step 0 — Type
  category: AppCategory;
  appType: AppType;
  exposure: AppExposure;
  command: string;
  cronSchedule: string;
  backgroundMode: "always" | "scheduled";
  // Step 1
  imageSource: "zenith" | "external";
  selectedImage: string;
  externalImage: string;
  isPrivateRegistry: boolean;
  regUser: string;
  regPass: string;
  // Step 2
  appName: string;
  port: string;
  portAutoDetected: boolean;
  envVars: EnvEntry[];
  // Step 3 (Resources)
  s3Enabled: boolean;
  s3Mode: "existing" | "new";
  s3Bucket: string;
  dbEnabled: boolean;
  dbMode: "existing" | "new";
  dbName: string;
  dbEngine: string;
}

const initialState: WizardState = {
  category: "frontend",
  appType: "web",
  exposure: "public",
  command: "",
  cronSchedule: "",
  backgroundMode: "always",
  imageSource: "external",
  selectedImage: "",
  externalImage: "",
  isPrivateRegistry: false,
  regUser: "",
  regPass: "",
  appName: "",
  port: "",
  portAutoDetected: false,
  envVars: [],
  s3Enabled: false,
  s3Mode: "existing",
  s3Bucket: "",
  dbEnabled: false,
  dbMode: "existing",
  dbName: "",
  dbEngine: "postgres",
};

// Steps are dynamic — background jobs skip the Resources step
const WEB_STEPS = ["Type", "Image", "Config", "Resources", "Review"] as const;
const BG_STEPS = ["Type", "Image", "Config", "Review"] as const;

// ── Helpers ──

function isValidAppName(name: string) {
  return /^[a-z][a-z0-9-]*$/.test(name) && !name.endsWith("-");
}

// ── Stepper ──

function Stepper({ current, completed, steps }: { current: number; completed: number[]; steps: readonly string[] }) {
  return (
    <div className="flex items-center justify-center gap-0 mb-6">
      {steps.map((label, i) => {
        const done = completed.includes(i);
        const active = i === current;
        return (
          <div key={label} className="flex items-center">
            {i > 0 && (
              <div className={`h-px w-10 ${done || active ? "bg-accent-500" : "bg-border"}`} />
            )}
            <div className="flex flex-col items-center gap-1">
              <div
                className={`flex h-7 w-7 items-center justify-center rounded-full border-2 text-xs font-semibold transition-colors ${
                  done
                    ? "border-accent-500 bg-accent-500 text-white"
                    : active
                      ? "border-accent-500 bg-transparent text-accent-400"
                      : "border-border bg-transparent text-neutral-500"
                }`}
              >
                {done ? <Check className="h-3.5 w-3.5" /> : i + 1}
              </div>
              <span className={`text-[10px] ${active ? "text-accent-400" : "text-neutral-500"}`}>
                {label}
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ── Main Component ──

interface DeployWizardProps {
  onClose: () => void;
  isPro: boolean;
  projectId: string;
}

export function DeployWizard({ onClose, isPro, projectId }: DeployWizardProps) {
  const { toast } = useToast();
  const { appsDeploy, registry, storage, databases } = getApi();
  const [step, setStep] = useState(0);
  const [state, setState] = useState<WizardState>(initialState);
  const [deploying, setDeploying] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [newEnvKey, setNewEnvKey] = useState("");
  const [newEnvValue, setNewEnvValue] = useState("");

  const isBackground = state.category === "background";
  const isCron = state.appType === "cron";
  const steps = isBackground ? BG_STEPS : WEB_STEPS;
  const lastStep = steps.length - 1;

  // Name availability check
  const [nameAvailable, setNameAvailable] = useState<boolean | null>(null);
  const [nameCheckUrl, setNameCheckUrl] = useState("");
  const [nameChecking, setNameChecking] = useState(false);

  // Debounced name availability check
  useEffect(() => {
    const name = state.appName.trim();
    if (!name || !isValidAppName(name)) {
      setNameAvailable(null);
      setNameCheckUrl("");
      return;
    }
    setNameChecking(true);
    const timer = setTimeout(() => {
      appsDeploy.checkName(name).then((res) => {
        setNameAvailable(res.available);
        setNameCheckUrl(res.url);
        setNameChecking(false);
      }).catch(() => {
        setNameAvailable(null);
        setNameChecking(false);
      });
    }, 400);
    return () => clearTimeout(timer);
  }, [state.appName, appsDeploy]);

  // Smart port detection when image changes
  useEffect(() => {
    const imageRef = state.imageSource === "zenith" ? state.selectedImage : state.externalImage;
    const detected = detectWellKnownImage(imageRef);
    if (detected && (state.port === "" || state.portAutoDetected)) {
      setState((s) => ({ ...s, port: String(detected.port), portAutoDetected: true }));
    }
  }, [state.externalImage, state.selectedImage, state.imageSource]);

  // Registry images (lazy-loaded when Zenith source selected)
  const [registryImages, setRegistryImages] = useState<RegistryImage[]>([]);
  const [registryLoading, setRegistryLoading] = useState(false);
  const [registrySearch, setRegistrySearch] = useState("");
  const [expandedImage, setExpandedImage] = useState<string | null>(null);

  // Existing S3 buckets (lazy-loaded when step 2 reached for Pro+)
  const [existingBuckets, setExistingBuckets] = useState<StorageBucket[]>([]);
  const [bucketsLoading, setBucketsLoading] = useState(false);
  const [bucketsFetched, setBucketsFetched] = useState(false);

  // Existing databases (lazy-loaded when step 2 reached)
  const [existingDbs, setExistingDbs] = useState<DbType[]>([]);
  const [dbsLoading, setDbsLoading] = useState(false);
  const [dbsFetched, setDbsFetched] = useState(false);

  useEffect(() => {
    if (state.imageSource === "zenith" && registryImages.length === 0 && !registryLoading) {
      setRegistryLoading(true);
      registry.listImages().then((res) => {
        setRegistryImages(res.items);
        setRegistryLoading(false);
      }).catch(() => setRegistryLoading(false));
    }
  }, [state.imageSource, registry, registryImages.length, registryLoading]);

  // Fetch existing buckets when entering Resources step as Pro+
  useEffect(() => {
    if (!isBackground && step === 3 && isPro && !bucketsFetched && !bucketsLoading && projectId) {
      setBucketsLoading(true);
      storage.list(projectId).then((res) => {
        setExistingBuckets(res.items);
        setBucketsFetched(true);
        setBucketsLoading(false);
      }).catch(() => {
        setBucketsFetched(true);
        setBucketsLoading(false);
      });
    }
  }, [step, isBackground, isPro, projectId, storage, bucketsFetched, bucketsLoading]);

  // Fetch existing databases when entering Resources step (all tiers)
  useEffect(() => {
    if (!isBackground && step === 3 && !dbsFetched && !dbsLoading && projectId) {
      setDbsLoading(true);
      databases.list(projectId).then((res) => {
        setExistingDbs(res.items);
        setDbsFetched(true);
        setDbsLoading(false);
      }).catch(() => {
        setDbsFetched(true);
        setDbsLoading(false);
      });
    }
  }, [step, isBackground, projectId, databases, dbsFetched, dbsLoading]);

  const update = <K extends keyof WizardState>(key: K, value: WizardState[K]) =>
    setState((s) => ({ ...s, [key]: value }));

  // Track the highest step the user has visited
  const [highestStep, setHighestStep] = useState(0);

  const goToStep = (s: number) => {
    setStep(s);
    setHighestStep((prev) => Math.max(prev, s));
  };

  const completedSteps = (() => {
    const c: number[] = [];
    // Step 0 (Type) always passable (has default)
    if (highestStep > 0) c.push(0);
    // Step 1 done if an image is selected
    const hasImage = state.imageSource === "zenith" ? !!state.selectedImage : !!state.externalImage.trim();
    if (hasImage) c.push(1);
    // Step 2 done if name valid
    if (state.appName.trim() && isValidAppName(state.appName.trim())) c.push(2);
    // Step 3: Resources (web/worker only) — optional, passing through = done
    if (!isBackground && highestStep > 3) c.push(3);
    return c;
  })();

  // ── Validation per step ──

  const canNext = (() => {
    if (step === 0) {
      // Type step: scheduled background needs a cron expression
      if (state.category === "background" && state.backgroundMode === "scheduled") return !!state.cronSchedule.trim();
      return true;
    }
    if (step === 1) {
      if (state.imageSource === "zenith") return !!state.selectedImage;
      const hasImg = !!state.externalImage.trim();
      if (state.isPrivateRegistry) return hasImg && !!state.regUser.trim() && !!state.regPass.trim();
      return hasImg;
    }
    if (step === 2) {
      return !!state.appName.trim() && isValidAppName(state.appName.trim());
    }
    return true; // resources / review always passable
  })();

  // ── Deploy handler ──

  const handleDeploy = async () => {
    setDeploying(true);
    try {
      const imageUrl =
        state.imageSource === "zenith"
          ? state.selectedImage
          : state.externalImage.trim();

      const port = parseInt(state.port, 10) || 0; // 0 = let API resolve from well-known images or default

      const envVarsToSend = state.envVars.filter((e) => e.key && e.value);

      await appsDeploy.create({
        name: state.appName.trim(),
        deploy_source: "image",
        image_url: imageUrl,
        ...(port > 0 && { port }),
        app_type: state.appType,
        exposure: state.exposure,
        ...(state.appType === "worker" && state.command && { command: state.command.trim() }),
        ...(state.appType === "cron" && state.cronSchedule && { cron_schedule: state.cronSchedule.trim() }),
        ...(state.imageSource === "external" &&
          state.isPrivateRegistry && {
            registry_username: state.regUser.trim(),
            registry_password: state.regPass.trim(),
          }),
        ...(envVarsToSend.length > 0 && { env_vars: envVarsToSend }),
      });

      toast("success", "Application deployed successfully! Build is starting...");
      onClose();
    } catch (err: unknown) {
      const apiErr = err as { status?: number; body?: { error?: string } };
      if (apiErr?.status === 409) {
        toast("error", "This app name is already taken. Please choose another.");
      } else {
        toast("error", "Failed to deploy application");
      }
    } finally {
      setDeploying(false);
    }
  };

  // ── Add env var ──

  const addEnvVar = () => {
    const k = newEnvKey.trim();
    const v = newEnvValue.trim();
    if (!k) return;
    // Prevent duplicates
    if (state.envVars.some((e) => e.key === k)) return;
    update("envVars", [...state.envVars, { key: k, value: v }]);
    setNewEnvKey("");
    setNewEnvValue("");
  };

  const removeEnvVar = (key: string) =>
    update("envVars", state.envVars.filter((e) => e.key !== key));

  // ── Filtered registry images ──

  const filteredImages = registrySearch
    ? registryImages.filter((img) =>
        img.name.toLowerCase().includes(registrySearch.toLowerCase())
      )
    : registryImages;

  // ── Step renderers ──

  // Helper to set category and derive appType + exposure
  const setCategory = (cat: AppCategory) => {
    if (cat === "frontend") {
      setState((s) => ({ ...s, category: cat, appType: "web", exposure: "public" }));
    } else if (cat === "api") {
      setState((s) => ({ ...s, category: cat, appType: "web", exposure: "protected" }));
    } else {
      // background — keep existing backgroundMode
      const bgType = state.backgroundMode === "scheduled" ? "cron" : "worker";
      setState((s) => ({ ...s, category: cat, appType: bgType as AppType, exposure: "public" }));
    }
  };

  const setBackgroundMode = (mode: "always" | "scheduled") => {
    setState((s) => ({
      ...s,
      backgroundMode: mode,
      appType: mode === "scheduled" ? "cron" : "worker",
    }));
  };

  const renderStepType = () => (
    <div className="space-y-4">
      <p className="text-sm text-neutral-400">What are you deploying?</p>
      <div className="grid gap-3 grid-cols-3">
        {([
          { cat: "frontend" as AppCategory, icon: Globe, label: "Frontend", desc: "Public web app with a URL. React, Next.js, Vue, nginx, dashboards." },
          { cat: "api" as AppCategory, icon: Shield, label: "API Service", desc: "Protected HTTP service behind the API gateway. Express, FastAPI, Go, Spring Boot." },
          { cat: "background" as AppCategory, icon: Cog, label: "Background Job", desc: "No URL, runs in the cluster. Workers, crons, data pipelines." },
        ] as const).map(({ cat, icon: Icon, label, desc }) => (
          <button
            key={cat}
            type="button"
            onClick={() => setCategory(cat)}
            className={`rounded-lg border p-4 text-left transition-colors ${
              state.category === cat
                ? "border-accent-500 bg-accent-500/10"
                : "border-border bg-surface-100 hover:border-neutral-600"
            }`}
          >
            <Icon className={`h-5 w-5 mb-2 ${state.category === cat ? "text-accent-400" : "text-neutral-500"}`} />
            <div className="text-sm font-medium text-white">{label}</div>
            <p className="mt-0.5 text-[11px] text-neutral-500">{desc}</p>
          </button>
        ))}
      </div>

      {/* Background: sub-options */}
      {state.category === "background" && (
        <div className="space-y-3">
          <p className="text-xs font-medium text-neutral-400">Run mode</p>
          <div className="grid gap-3 grid-cols-2">
            <button
              type="button"
              onClick={() => setBackgroundMode("always")}
              className={`rounded-lg border p-3 text-left transition-colors ${
                state.backgroundMode === "always"
                  ? "border-accent-500 bg-accent-500/10"
                  : "border-border bg-surface-100 hover:border-neutral-600"
              }`}
            >
              <Cog className={`h-4 w-4 mb-1 ${state.backgroundMode === "always" ? "text-accent-400" : "text-neutral-500"}`} />
              <div className="text-sm font-medium text-white">Always Running</div>
              <p className="mt-0.5 text-[11px] text-neutral-500">Queue workers, bots, data processors</p>
            </button>
            <button
              type="button"
              onClick={() => setBackgroundMode("scheduled")}
              className={`rounded-lg border p-3 text-left transition-colors ${
                state.backgroundMode === "scheduled"
                  ? "border-accent-500 bg-accent-500/10"
                  : "border-border bg-surface-100 hover:border-neutral-600"
              }`}
            >
              <Clock className={`h-4 w-4 mb-1 ${state.backgroundMode === "scheduled" ? "text-accent-400" : "text-neutral-500"}`} />
              <div className="text-sm font-medium text-white">Scheduled</div>
              <p className="mt-0.5 text-[11px] text-neutral-500">Backups, reports, cleanup scripts</p>
            </button>
          </div>
        </div>
      )}

      {/* Always-running worker: optional command */}
      {state.category === "background" && state.backgroundMode === "always" && (
        <div>
          <label className="mb-1.5 block text-xs font-medium text-neutral-400">
            Command Override <span className="text-neutral-600">(optional)</span>
          </label>
          <input
            type="text"
            value={state.command}
            onChange={(e) => update("command", e.target.value)}
            placeholder="e.g. npm run worker"
            className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
          />
        </div>
      )}

      {/* Scheduled cron: required schedule + optional command */}
      {state.category === "background" && state.backgroundMode === "scheduled" && (
        <div className="space-y-3">
          <div>
            <label className="mb-1.5 block text-xs font-medium text-neutral-400">
              Cron Schedule <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              value={state.cronSchedule}
              onChange={(e) => update("cronSchedule", e.target.value)}
              placeholder="0 6 * * *"
              className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm font-mono text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
            />
            <p className="mt-1 text-[11px] text-neutral-600">
              Standard cron syntax: minute hour day month weekday
            </p>
          </div>
          <div>
            <label className="mb-1.5 block text-xs font-medium text-neutral-400">
              Command Override <span className="text-neutral-600">(optional)</span>
            </label>
            <input
              type="text"
              value={state.command}
              onChange={(e) => update("command", e.target.value)}
              placeholder="e.g. python report.py"
              className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
            />
          </div>
        </div>
      )}
    </div>
  );

  const renderStep0 = () => (
    <div className="space-y-4">
      {/* Source selector */}
      <div className="grid gap-3 grid-cols-2">
        {isPro && (
          <button
            type="button"
            onClick={() => update("imageSource", "zenith")}
            className={`rounded-lg border p-4 text-left transition-colors ${
              state.imageSource === "zenith"
                ? "border-accent-500 bg-accent-500/10"
                : "border-border bg-surface-100 hover:border-neutral-600"
            }`}
          >
            <div className="flex items-center gap-2 mb-1">
              <Crown className="h-4 w-4 text-amber-400" />
              <span className="text-sm font-medium text-white">Zenith Registry</span>
            </div>
            <p className="text-[11px] text-neutral-500">
              Browse images from your built-in registry
            </p>
          </button>
        )}
        <button
          type="button"
          onClick={() => update("imageSource", "external")}
          className={`rounded-lg border p-4 text-left transition-colors ${
            state.imageSource === "external"
              ? "border-accent-500 bg-accent-500/10"
              : "border-border bg-surface-100 hover:border-neutral-600"
          } ${!isPro ? "col-span-2" : ""}`}
        >
          <div className="flex items-center gap-2 mb-1">
            <Container className="h-4 w-4 text-neutral-400" />
            <span className="text-sm font-medium text-white">External Registry</span>
          </div>
          <p className="text-[11px] text-neutral-500">
            Docker Hub, GHCR, or any container registry
          </p>
        </button>
      </div>

      {!isPro && (
        <p className="flex items-center gap-1.5 text-[11px] text-neutral-500">
          <Crown className="h-3 w-3 text-amber-400" />
          Upgrade to Pro for Zenith&apos;s built-in registry with image browsing
        </p>
      )}

      {/* Zenith registry browser */}
      {state.imageSource === "zenith" && isPro && (
        <div className="space-y-3">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500" />
            <input
              type="text"
              value={registrySearch}
              onChange={(e) => setRegistrySearch(e.target.value)}
              placeholder="Search images..."
              className="w-full rounded-lg border border-border bg-surface-200 py-2 pl-9 pr-3 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
            />
          </div>

          {registryLoading ? (
            <div className="flex items-center justify-center py-8 text-neutral-500">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : filteredImages.length === 0 ? (
            <p className="py-6 text-center text-sm text-neutral-500">No images found</p>
          ) : (
            <div className="max-h-64 space-y-2 overflow-y-auto">
              {filteredImages.map((img) => {
                const isExpanded = expandedImage === img.name;
                const isSelected = state.selectedImage.startsWith(`hub.stage.freezenith.com/${img.name}:`);
                const selectedTag = isSelected
                  ? state.selectedImage.split(":").pop()
                  : null;

                return (
                  <div key={img.name} className={`rounded-lg border transition-colors ${
                    isSelected
                      ? "border-accent-500 bg-accent-500/10"
                      : "border-border bg-surface-100"
                  }`}>
                    <button
                      type="button"
                      onClick={() => setExpandedImage(isExpanded ? null : img.name)}
                      className="flex w-full items-center justify-between p-3 text-left"
                    >
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-white">{img.name}</span>
                          {selectedTag && (
                            <span className="rounded bg-accent-500/20 px-1.5 py-0.5 text-[10px] font-mono text-accent-400">
                              :{selectedTag}
                            </span>
                          )}
                        </div>
                        <div className="mt-1 flex items-center gap-3 text-[11px] text-neutral-500">
                          <span>{img.tags.length} tag{img.tags.length !== 1 ? "s" : ""}</span>
                          <span>{img.size}</span>
                          <span>pushed {img.lastPushed}</span>
                        </div>
                      </div>
                      <ChevronDown className={`h-4 w-4 text-neutral-500 transition-transform ${isExpanded ? "rotate-180" : ""}`} />
                    </button>

                    {isExpanded && (
                      <div className="border-t border-border px-3 pb-3 pt-2 space-y-1">
                        {img.tags.map((tag) => {
                          const fullRef = `hub.stage.freezenith.com/${img.name}:${tag}`;
                          const isTagSelected = state.selectedImage === fullRef;
                          return (
                            <button
                              key={tag}
                              type="button"
                              onClick={() => update("selectedImage", fullRef)}
                              className={`flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors ${
                                isTagSelected
                                  ? "bg-accent-500/15 text-accent-400"
                                  : "text-neutral-400 hover:bg-surface-300 hover:text-white"
                              }`}
                            >
                              <Tag className="h-3 w-3 shrink-0" />
                              <span className="font-mono">{tag}</span>
                              {isTagSelected && <Check className="ml-auto h-3 w-3 text-accent-400" />}
                            </button>
                          );
                        })}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* External image input */}
      {state.imageSource === "external" && (
        <div className="space-y-4">
          <div>
            <label className="mb-1.5 block text-xs font-medium text-neutral-400">
              Container Image
            </label>
            <div className="relative">
              <Container className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500" />
              <input
                type="text"
                value={state.externalImage}
                onChange={(e) => update("externalImage", e.target.value)}
                placeholder="nginx, user/app, or ghcr.io/org/image:tag"
                className="w-full rounded-lg border border-border bg-surface-200 py-2.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            {state.externalImage.trim() && (() => {
              const resolved = normalizeImageRef(state.externalImage.trim());
              const changed = resolved !== state.externalImage.trim();
              const detected = detectWellKnownImage(state.externalImage.trim());
              return (
                <div className="mt-1 space-y-0.5">
                  {detected && (
                    <p className="text-[11px] text-emerald-400">
                      Detected: <span className="font-medium">{detected.description}</span> (port {detected.port})
                    </p>
                  )}
                  {changed && (
                    <p className="text-[11px] text-neutral-500">
                      Resolves to <span className="font-mono text-accent-400">{resolved}</span>
                    </p>
                  )}
                </div>
              );
            })()}
          </div>

          {/* Private registry toggle */}
          <div>
            <button
              type="button"
              onClick={() => update("isPrivateRegistry", !state.isPrivateRegistry)}
              className="flex items-center gap-2 text-sm text-neutral-400 hover:text-neutral-300 transition-colors"
            >
              <div
                className={`flex h-4 w-7 items-center rounded-full transition-colors ${
                  state.isPrivateRegistry
                    ? "bg-accent-500 justify-end"
                    : "bg-surface-300 justify-start"
                }`}
              >
                <div className="mx-0.5 h-3 w-3 rounded-full bg-white" />
              </div>
              <Lock className="h-3.5 w-3.5" />
              Private registry
            </button>
          </div>

          {state.isPrivateRegistry && (
            <div className="space-y-3 rounded-lg border border-border bg-surface-100 p-3">
              <div>
                <label className="mb-1.5 block text-xs font-medium text-neutral-400">
                  Username
                </label>
                <input
                  type="text"
                  value={state.regUser}
                  onChange={(e) => update("regUser", e.target.value)}
                  placeholder="registry username"
                  className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-xs font-medium text-neutral-400">
                  Password
                </label>
                <div className="relative">
                  <input
                    type={showPassword ? "text" : "password"}
                    value={state.regPass}
                    onChange={(e) => update("regPass", e.target.value)}
                    placeholder="registry password or token"
                    className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 pr-10 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-500 hover:text-neutral-300"
                  >
                    {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );

  const renderStep1 = () => {
    const planKey = isPro ? "pro" : "free";
    const res = PLAN_RESOURCES[planKey];
    return (
    <div className="space-y-4">
      {/* App Name */}
      <div>
        <label className="mb-1.5 block text-xs font-medium text-neutral-400">
          App Name
        </label>
        <input
          type="text"
          value={state.appName}
          onChange={(e) => update("appName", e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
          placeholder="my-app"
          className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
        />
        {state.appName && !isValidAppName(state.appName) ? (
          <p className="mt-1 text-[11px] text-red-400">
            Must start with a letter and contain only lowercase letters, numbers, and hyphens
          </p>
        ) : state.appName && isValidAppName(state.appName) ? (
          <div className="mt-1 space-y-0.5">
            {nameChecking ? (
              <p className="text-[11px] text-neutral-500">Checking availability...</p>
            ) : nameAvailable === true && nameCheckUrl ? (
              <p className="text-[11px] text-emerald-400">
                <Check className="inline h-3 w-3 mr-0.5" />
                Available &mdash; <span className="font-mono text-emerald-400/80">{nameCheckUrl}</span>
              </p>
            ) : nameAvailable === false ? (
              <p className="text-[11px] text-red-400">
                This name is taken, try another
              </p>
            ) : (
              <p className="text-[11px] text-neutral-600">
                Lowercase letters, numbers, and hyphens only
              </p>
            )}
          </div>
        ) : (
          <p className="mt-1 text-[11px] text-neutral-600">
            Lowercase letters, numbers, and hyphens only
          </p>
        )}
      </div>

      {/* Port (web only) */}
      {state.appType === "web" && (
        <div>
          <label className="mb-1.5 block text-xs font-medium text-neutral-400">
            Port
          </label>
          <input
            type="number"
            value={state.port}
            onChange={(e) => {
              update("port", e.target.value);
              setState((s) => ({ ...s, port: e.target.value, portAutoDetected: false }));
            }}
            placeholder="8080"
            min={1}
            max={65535}
            className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
          />
          {state.portAutoDetected ? (
            <p className="mt-1 text-[11px] text-emerald-400">
              Auto-detected from image. You can override this.
            </p>
          ) : (
            <p className="mt-1 text-[11px] text-neutral-600">
              The port your application listens on{!state.port && " (defaults to 8080)"}
            </p>
          )}
        </div>
      )}

      {/* Plan Resources (read-only info) */}
      <div>
        <label className="mb-1.5 block text-xs font-medium text-neutral-400">
          Resources
        </label>
        <div className="rounded-lg border border-border bg-surface-100 p-3">
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1.5 text-sm text-white">
              <Cpu className="h-3.5 w-3.5 text-neutral-500" />
              {res.cpu}
            </div>
            <div className="text-neutral-600">|</div>
            <div className="text-sm text-white">{res.ram}</div>
          </div>
          <p className="mt-1.5 text-[11px] text-neutral-500">
            {isPro ? "Pro plan" : "Free plan"} per-app limit.
            {!isPro && (
              <span className="ml-1 text-amber-400">Upgrade to Pro for more resources.</span>
            )}
          </p>
        </div>
      </div>

      {/* Env Vars */}
      <div>
        <label className="mb-1.5 block text-xs font-medium text-neutral-400">
          Environment Variables
        </label>

        {/* Add row */}
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={newEnvKey}
            onChange={(e) => setNewEnvKey(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, ""))}
            placeholder="KEY"
            className="w-1/3 rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm font-mono text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
          />
          <input
            type="text"
            value={newEnvValue}
            onChange={(e) => setNewEnvValue(e.target.value)}
            placeholder="value"
            className="flex-1 rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
          />
          <button
            type="button"
            onClick={addEnvVar}
            disabled={!newEnvKey.trim()}
            className="rounded-lg border border-border p-2 text-neutral-400 hover:text-accent-400 hover:border-accent-500 transition-colors disabled:opacity-30 disabled:hover:text-neutral-400 disabled:hover:border-border"
          >
            <Plus className="h-4 w-4" />
          </button>
        </div>

        {/* Existing vars */}
        {state.envVars.length > 0 && (
          <div className="mt-3 space-y-1.5">
            {state.envVars.map((env) => (
              <div
                key={env.key}
                className="flex items-center justify-between rounded-lg border border-border bg-surface-100 px-3 py-2"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <span className="font-mono text-xs text-accent-400 shrink-0">{env.key}</span>
                  <span className="text-xs text-neutral-500 truncate">{env.value || "(empty)"}</span>
                </div>
                <button
                  type="button"
                  onClick={() => removeEnvVar(env.key)}
                  className="text-neutral-500 hover:text-red-400 transition-colors shrink-0 ml-2"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
  };

  const renderStep2 = () => (
    <div className="space-y-6">
      <p className="text-sm text-neutral-400">
        Attach resources to your app. Zenith handles credentials automatically
        &mdash; just read the environment variables in your code. No config files needed.
      </p>

      {/* ── Database Section (all tiers) ── */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={() => update("dbEnabled", !state.dbEnabled)}
            className="flex items-center gap-2 text-sm text-neutral-300 hover:text-white transition-colors"
          >
            <div
              className={`flex h-4 w-7 items-center rounded-full transition-colors ${
                state.dbEnabled ? "bg-accent-500 justify-end" : "bg-surface-300 justify-start"
              }`}
            >
              <div className="mx-0.5 h-3 w-3 rounded-full bg-white" />
            </div>
            <Database className="h-3.5 w-3.5" />
            Attach Database
          </button>
        </div>
        {!state.dbEnabled && (
          <p className="text-[11px] text-neutral-600 ml-9">
            Managed PostgreSQL, MySQL, or Redis. Credentials are generated and injected
            as <span className="font-mono text-neutral-500">DATABASE_URL</span> automatically.
          </p>
        )}

        {state.dbEnabled && (
          <div className="space-y-3 rounded-lg border border-border bg-surface-100 p-4">
            {/* Mode tabs */}
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => { update("dbMode", "existing"); update("dbName", ""); }}
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                  state.dbMode === "existing"
                    ? "bg-accent-500/15 text-accent-400"
                    : "text-neutral-400 hover:text-white"
                }`}
              >
                Existing database
              </button>
              <button
                type="button"
                onClick={() => { update("dbMode", "new"); update("dbName", ""); }}
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                  state.dbMode === "new"
                    ? "bg-accent-500/15 text-accent-400"
                    : "text-neutral-400 hover:text-white"
                }`}
              >
                Create new
              </button>
            </div>

            {state.dbMode === "existing" ? (
              <div>
                {dbsLoading ? (
                  <div className="flex items-center justify-center py-6 text-neutral-500">
                    <Loader2 className="h-5 w-5 animate-spin" />
                  </div>
                ) : existingDbs.length === 0 ? (
                  <div className="rounded-lg border border-border bg-surface-200 px-4 py-6 text-center">
                    <p className="text-sm text-neutral-500">No databases yet</p>
                    <button
                      type="button"
                      onClick={() => update("dbMode", "new")}
                      className="mt-2 text-xs text-accent-400 hover:text-accent-300 transition-colors"
                    >
                      Create a new database
                    </button>
                  </div>
                ) : (
                  <div className="space-y-2 max-h-40 overflow-y-auto">
                    {existingDbs.map((db) => (
                      <button
                        key={db.name}
                        type="button"
                        onClick={() => {
                          update("dbName", db.name);
                          update("dbEngine", db.engine);
                        }}
                        className={`w-full rounded-lg border p-3 text-left transition-colors ${
                          state.dbName === db.name
                            ? "border-accent-500 bg-accent-500/10"
                            : "border-border bg-surface-200 hover:border-neutral-600"
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium text-white">{db.name}</span>
                          <span className={`text-[11px] ${db.status === "running" ? "text-emerald-400" : "text-neutral-500"}`}>
                            {db.status}
                          </span>
                        </div>
                        <div className="mt-1 flex items-center gap-3 text-[11px] text-neutral-500">
                          <span>{db.engine} {db.version}</span>
                          <span>{db.storage}</span>
                        </div>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <div className="space-y-3">
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-neutral-400">Engine</label>
                  <select
                    value={state.dbEngine}
                    onChange={(e) => update("dbEngine", e.target.value)}
                    className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                  >
                    <option value="postgres">PostgreSQL</option>
                    <option value="mysql">MySQL</option>
                    <option value="redis">Redis</option>
                  </select>
                </div>
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-neutral-400">Database Name</label>
                  <input
                    type="text"
                    value={state.dbName}
                    onChange={(e) => update("dbName", e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                    placeholder="my-app-db"
                    className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                  />
                </div>
              </div>
            )}

            <div className="rounded-md bg-surface-200 px-3 py-2.5 space-y-2">
              <div>
                <p className="text-[10px] font-medium text-neutral-500 mb-1">Auto-injected env var:</p>
                <p className="font-mono text-[11px] text-accent-400/80">
                  DATABASE_URL=<span className="text-neutral-600">{state.dbEngine === "redis" ? "redis" : "postgres"}://user:****@host:{state.dbEngine === "redis" ? "6379" : "5432"}/{state.dbName || "dbname"}</span>
                </p>
              </div>
              <div>
                <p className="text-[10px] font-medium text-neutral-500 mb-1">Use in your code:</p>
                <p className="font-mono text-[11px] text-neutral-500">
                  {state.dbEngine === "redis"
                    ? "redis.createClient({ url: process.env.DATABASE_URL })"
                    : "pg.connect(process.env.DATABASE_URL)"}
                </p>
              </div>
              <p className="text-[10px] text-neutral-600">
                Credentials are generated securely. You never need to manage passwords.
              </p>
            </div>
          </div>
        )}
      </div>

      {/* ── S3 Section (Pro+ only) ── */}
      {isPro ? (
        <div className="space-y-3">
          <button
            type="button"
            onClick={() => update("s3Enabled", !state.s3Enabled)}
            className="flex items-center gap-2 text-sm text-neutral-300 hover:text-white transition-colors"
          >
            <div
              className={`flex h-4 w-7 items-center rounded-full transition-colors ${
                state.s3Enabled ? "bg-accent-500 justify-end" : "bg-surface-300 justify-start"
              }`}
            >
              <div className="mx-0.5 h-3 w-3 rounded-full bg-white" />
            </div>
            <HardDrive className="h-3.5 w-3.5" />
            Attach S3 Bucket
          </button>
          {!state.s3Enabled && (
            <p className="text-[11px] text-neutral-600 ml-9">
              S3-compatible object storage for files, images, and uploads. Access it from your code
              using any S3 SDK &mdash; no volume mounts needed.
            </p>
          )}

          {state.s3Enabled && (
            <div className="space-y-3 rounded-lg border border-border bg-surface-100 p-4">
              {/* Mode tabs */}
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => { update("s3Mode", "existing"); update("s3Bucket", ""); }}
                  className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                    state.s3Mode === "existing"
                      ? "bg-accent-500/15 text-accent-400"
                      : "text-neutral-400 hover:text-white"
                  }`}
                >
                  Existing bucket
                </button>
                <button
                  type="button"
                  onClick={() => { update("s3Mode", "new"); update("s3Bucket", ""); }}
                  className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                    state.s3Mode === "new"
                      ? "bg-accent-500/15 text-accent-400"
                      : "text-neutral-400 hover:text-white"
                  }`}
                >
                  Create new
                </button>
              </div>

              {state.s3Mode === "existing" ? (
                <div>
                  {bucketsLoading ? (
                    <div className="flex items-center justify-center py-6 text-neutral-500">
                      <Loader2 className="h-5 w-5 animate-spin" />
                    </div>
                  ) : existingBuckets.length === 0 ? (
                    <div className="rounded-lg border border-border bg-surface-200 px-4 py-6 text-center">
                      <p className="text-sm text-neutral-500">No buckets yet</p>
                      <button
                        type="button"
                        onClick={() => update("s3Mode", "new")}
                        className="mt-2 text-xs text-accent-400 hover:text-accent-300 transition-colors"
                      >
                        Create a new bucket
                      </button>
                    </div>
                  ) : (
                    <div className="space-y-2 max-h-40 overflow-y-auto">
                      {existingBuckets.map((bucket) => (
                        <button
                          key={bucket.name}
                          type="button"
                          onClick={() => update("s3Bucket", bucket.name)}
                          className={`w-full rounded-lg border p-3 text-left transition-colors ${
                            state.s3Bucket === bucket.name
                              ? "border-accent-500 bg-accent-500/10"
                              : "border-border bg-surface-200 hover:border-neutral-600"
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <span className="text-sm font-medium text-white">{bucket.name}</span>
                            <span className="text-[11px] text-neutral-500">{bucket.size}</span>
                          </div>
                          <div className="mt-1 flex items-center gap-3 text-[11px] text-neutral-500">
                            <span>{bucket.objects} object{bucket.objects !== 1 ? "s" : ""}</span>
                            <span>{bucket.status}</span>
                          </div>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              ) : (
                <div>
                  <input
                    type="text"
                    value={state.s3Bucket}
                    onChange={(e) => update("s3Bucket", e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                    placeholder="my-app-uploads"
                    className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                  />
                </div>
              )}

              <div className="rounded-md bg-surface-200 px-3 py-2.5 space-y-2">
                <div>
                  <p className="text-[10px] font-medium text-neutral-500 mb-1">Auto-injected env vars:</p>
                  <div className="font-mono text-[11px] space-y-0.5">
                    <p><span className="text-accent-400/80">S3_ENDPOINT</span><span className="text-neutral-600">=https://fsn1.your-objectstorage.com</span></p>
                    <p><span className="text-accent-400/80">S3_BUCKET</span><span className="text-neutral-600">={state.s3Bucket || "bucket-name"}</span></p>
                    <p><span className="text-accent-400/80">S3_ACCESS_KEY</span><span className="text-neutral-600">=****</span></p>
                    <p><span className="text-accent-400/80">S3_SECRET_KEY</span><span className="text-neutral-600">=****</span></p>
                  </div>
                </div>
                <div>
                  <p className="text-[10px] font-medium text-neutral-500 mb-1">Use in your code:</p>
                  <p className="font-mono text-[11px] text-neutral-500">
                    s3.putObject(&#123; Bucket: process.env.S3_BUCKET, ... &#125;)
                  </p>
                </div>
                <p className="text-[10px] text-neutral-600">
                  Works with any S3-compatible SDK (aws-sdk, boto3, minio-go). No volume mounts &mdash; access files via API.
                </p>
              </div>
            </div>
          )}
        </div>
      ) : (
        <div className="rounded-lg border border-border bg-surface-100 px-4 py-3">
          <div className="flex items-center gap-2">
            <Crown className="h-4 w-4 text-amber-400 shrink-0" />
            <p className="text-xs text-neutral-400">S3 Object Storage</p>
          </div>
          <p className="mt-1 ml-6 text-[11px] text-neutral-600">
            Upgrade to Pro to attach S3 buckets for file uploads, images, and static assets.
            Access via any S3 SDK &mdash; credentials are injected automatically.
          </p>
        </div>
      )}
    </div>
  );

  const renderStep3 = () => {
    const imageUrl =
      state.imageSource === "zenith" ? state.selectedImage : state.externalImage.trim();
    const categoryLabels: Record<AppCategory, string> = { frontend: "Frontend", api: "API Service", background: "Background Job" };
    const planKey = isPro ? "pro" : "free";
    const planRes = PLAN_RESOURCES[planKey];

    return (
      <div className="space-y-4">
        {/* Image */}
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h3 className="text-xs font-medium text-neutral-500 mb-2">Image</h3>
          <p className="text-sm font-mono text-white break-all">{imageUrl}</p>
          {state.imageSource === "external" && state.isPrivateRegistry && (
            <p className="mt-1 text-[11px] text-neutral-500">
              Private registry: {state.regUser}
            </p>
          )}
        </div>

        {/* Config */}
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h3 className="text-xs font-medium text-neutral-500 mb-2">Configuration</h3>
          <div className="grid grid-cols-2 gap-y-2 text-sm">
            <span className="text-neutral-400">Type</span>
            <span className="text-white">{categoryLabels[state.category]}{state.category === "background" ? ` (${state.backgroundMode === "scheduled" ? "Scheduled" : "Always Running"})` : ""}</span>
            <span className="text-neutral-400">Name</span>
            <span className="text-white font-mono">{state.appName}</span>
            {state.category !== "background" && (
              <>
                <span className="text-neutral-400">Exposure</span>
                <span className="text-white">{state.exposure === "protected" ? "Protected (JWT)" : "Public"}</span>
              </>
            )}
            {state.category !== "background" && (
              <>
                <span className="text-neutral-400">URL</span>
                <span className="text-white font-mono text-xs">{nameCheckUrl || `https://${state.appName}.apps.stage.freezenith.com`}</span>
              </>
            )}
            {state.category !== "background" && (
              <>
                <span className="text-neutral-400">Port</span>
                <span className="text-white font-mono">{state.port || "auto-detect"}</span>
              </>
            )}
            {isCron && (
              <>
                <span className="text-neutral-400">Schedule</span>
                <span className="text-white font-mono">{state.cronSchedule}</span>
              </>
            )}
            {state.command && (
              <>
                <span className="text-neutral-400">Command</span>
                <span className="text-white font-mono text-xs">{state.command}</span>
              </>
            )}
            <span className="text-neutral-400">Resources</span>
            <span className="text-white text-xs">
              {planRes.cpu}, {planRes.ram}
              <span className="ml-1.5 text-neutral-500">({isPro ? "Pro" : "Free"} plan)</span>
            </span>
          </div>
        </div>

        {/* Env vars */}
        {state.envVars.length > 0 && (
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <h3 className="text-xs font-medium text-neutral-500 mb-2">
              Environment Variables ({state.envVars.length})
            </h3>
            <div className="space-y-1">
              {state.envVars.map((env) => (
                <div key={env.key} className="flex items-center gap-2 text-sm">
                  <span className="font-mono text-xs text-accent-400">{env.key}</span>
                  <span className="text-neutral-500">=</span>
                  <span className="font-mono text-xs text-neutral-400">{"*".repeat(Math.min(env.value.length || 3, 12))}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Resources (web/worker only) */}
        {!isBackground && ((state.dbEnabled && state.dbName) || (state.s3Enabled && state.s3Bucket)) ? (
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <h3 className="text-xs font-medium text-neutral-500 mb-2">Resources</h3>
            <div className="space-y-2">
              {state.dbEnabled && state.dbName && (
                <div className="flex items-center gap-2 text-sm">
                  <Database className="h-3.5 w-3.5 text-neutral-500" />
                  <span className="text-neutral-400">Database:</span>
                  <span className="text-white font-mono">{state.dbName}</span>
                  <span className="text-[11px] text-neutral-600">
                    ({state.dbMode === "new" ? `new ${state.dbEngine}` : "existing"})
                  </span>
                </div>
              )}
              {state.s3Enabled && state.s3Bucket && (
                <div className="flex items-center gap-2 text-sm">
                  <HardDrive className="h-3.5 w-3.5 text-neutral-500" />
                  <span className="text-neutral-400">S3 Bucket:</span>
                  <span className="text-white font-mono">{state.s3Bucket}</span>
                  <span className="text-[11px] text-neutral-600">
                    ({state.s3Mode === "new" ? "new" : "existing"})
                  </span>
                </div>
              )}
            </div>
          </div>
        ) : null}
      </div>
    );
  };

  // ── Render ──

  const stepContent = isBackground
    ? [renderStepType, renderStep0, renderStep1, renderStep3]
    : [renderStepType, renderStep0, renderStep1, renderStep2, renderStep3];

  return (
    <Modal title="Deploy App" onClose={onClose} size="lg">
      <Stepper current={step} completed={completedSteps} steps={steps} />

      {stepContent[step]()}

      {/* Footer nav */}
      <div className="mt-6 flex items-center justify-between border-t border-border pt-4">
        <div>
          {step > 0 && (
            <button
              type="button"
              onClick={() => goToStep(step - 1)}
              className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Back
            </button>
          )}
        </div>

        <div className="flex items-center gap-2">
          {!isBackground && step === 3 && (
            <button
              type="button"
              onClick={() => goToStep(step + 1)}
              className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Skip
            </button>
          )}

          {step < lastStep && (
            <button
              type="button"
              onClick={() => goToStep(step + 1)}
              disabled={!canNext}
              className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
            >
              Next
            </button>
          )}

          {step === lastStep && (
            <button
              type="button"
              onClick={handleDeploy}
              disabled={deploying}
              className="flex items-center gap-2 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-70"
            >
              {deploying ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Rocket className="h-4 w-4" />
              )}
              {deploying ? "Deploying..." : "Deploy"}
            </button>
          )}
        </div>
      </div>
    </Modal>
  );
}

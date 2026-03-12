"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Rocket, User, Server, Target, Code, Megaphone, PartyPopper, ChevronLeft } from "lucide-react";
import { getApi } from "@/lib/get-api";

const steps = [
  { title: "Welcome", icon: Rocket },
  { title: "About You", icon: User },
  { title: "Current Setup", icon: Server },
  { title: "Goals", icon: Target },
  { title: "Stack", icon: Code },
  { title: "Discovery", icon: Megaphone },
  { title: "Ready", icon: PartyPopper },
];

type Answers = {
  use_case: string;
  role: string;
  team_size: string;
  company_name: string;
  current_provider: string;
  monthly_spend: string;
  biggest_pain: string;
  expected_traffic: string;
  timeline: string;
  most_important: string;
  stack: string[];
  discovery: string;
};

function OptionButton({ selected, label, onClick }: { selected: boolean; label: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`w-full rounded-lg border px-4 py-2.5 text-left text-sm transition-colors ${
        selected
          ? "border-accent-500 bg-accent-500/10 text-accent-400"
          : "border-border bg-surface-100 text-neutral-300 hover:border-neutral-600"
      }`}
    >
      {label}
    </button>
  );
}

function MultiOptionButton({ selected, label, onClick }: { selected: boolean; label: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`rounded-lg border px-3 py-2 text-sm transition-colors ${
        selected
          ? "border-accent-500 bg-accent-500/10 text-accent-400"
          : "border-border bg-surface-100 text-neutral-300 hover:border-neutral-600"
      }`}
    >
      {label}
    </button>
  );
}

export default function OnboardingPage() {
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [userName, setUserName] = useState("");
  const [answers, setAnswers] = useState<Answers>({
    use_case: "",
    role: "",
    team_size: "",
    company_name: "",
    current_provider: "",
    monthly_spend: "",
    biggest_pain: "",
    expected_traffic: "",
    timeline: "",
    most_important: "",
    stack: [],
    discovery: "",
  });
  const { onboarding } = getApi();

  useEffect(() => {
    onboarding.getMe().then((data) => {
      if (data?.onboarding_completed) {
        router.replace("/");
        return;
      }
      setUserName(data?.name || "");
    }).catch(() => {
      router.replace("/login");
    });
  }, []);

  const trackStep = async (s: number, completed = false) => {
    try {
      if (completed) {
        await onboarding.update(s, true, answers as unknown as Record<string, unknown>);
      } else {
        await onboarding.update(s, false);
      }
    } catch {
      // non-blocking
    }
  };

  const canProceed = (): boolean => {
    switch (step) {
      case 0: return answers.use_case !== "";
      case 1: return answers.role !== "" && answers.team_size !== "";
      case 2: return answers.current_provider !== "" && answers.monthly_spend !== "";
      case 3: return answers.biggest_pain !== "" && answers.timeline !== "";
      case 4: return answers.stack.length > 0;
      case 5: return answers.discovery !== "";
      case 6: return true;
      default: return false;
    }
  };

  const finish = async () => {
    await trackStep(steps.length, true);
    router.push("/");
  };

  const next = async () => {
    if (!canProceed()) return;
    const nextStep = step + 1;
    if (nextStep >= steps.length) {
      await finish();
      return;
    }
    await trackStep(nextStep);
    setStep(nextStep);
  };

  const prev = () => {
    if (step > 0) setStep(step - 1);
  };

  const toggleStack = (val: string) => {
    setAnswers((prev) => ({
      ...prev,
      stack: prev.stack.includes(val)
        ? prev.stack.filter((s) => s !== val)
        : [...prev.stack, val],
    }));
  };

  return (
    <div className="min-h-screen bg-neutral-950 flex items-center justify-center p-4">
      <div className="w-full max-w-lg rounded-xl border border-border bg-surface-50 shadow-2xl">
        {/* Header — step indicators */}
        <div className="flex items-center justify-center border-b border-border px-6 py-4">
          <div className="flex items-center gap-1.5">
            {steps.map((s, i) => (
              <div key={i} className="flex items-center gap-1.5">
                <div
                  className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold transition-colors ${
                    i === step
                      ? "bg-accent-500 text-white"
                      : i < step
                        ? "bg-accent-500/20 text-accent-400"
                        : "bg-surface-200 text-neutral-600"
                  }`}
                >
                  {i + 1}
                </div>
                {i < steps.length - 1 && (
                  <div className={`h-px w-3 ${i < step ? "bg-accent-500/40" : "bg-surface-200"}`} />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="px-6 py-8 max-h-[65vh] overflow-y-auto">
          {/* Step 1: What brings you here? */}
          {step === 0 && (
            <div className="text-center space-y-4">
              <Rocket className="mx-auto h-12 w-12 text-accent-400" />
              <h2 className="text-xl font-semibold text-white">Welcome to Zenith{userName ? `, ${userName}` : ""}!</h2>
              <p className="text-sm text-neutral-400">Help us personalize your experience.</p>
              <div className="mt-6 space-y-2 text-left">
                <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">What brings you here?</p>
                {[
                  { value: "side_project", label: "Ship my side project / portfolio" },
                  { value: "saas", label: "Build and launch a SaaS product" },
                  { value: "startup", label: "Deploy for my startup / company" },
                  { value: "agency", label: "Host client projects (agency / freelancer)" },
                  { value: "learn", label: "Learn cloud-native and Kubernetes" },
                  { value: "migrate", label: "Migrate from another platform" },
                  { value: "evaluate", label: "Evaluating PaaS options for my team" },
                ].map((opt) => (
                  <OptionButton
                    key={opt.value}
                    selected={answers.use_case === opt.value}
                    label={opt.label}
                    onClick={() => setAnswers({ ...answers, use_case: opt.value })}
                  />
                ))}
              </div>
            </div>
          )}

          {/* Step 2: About you — role + team size + company */}
          {step === 1 && (
            <div className="text-center space-y-4">
              <User className="mx-auto h-12 w-12 text-blue-400" />
              <h2 className="text-xl font-semibold text-white">Tell us about yourself</h2>
              <p className="text-sm text-neutral-400">This helps us tailor features and support for you.</p>
              <div className="mt-6 space-y-5 text-left">
                <div className="space-y-2">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Your role</p>
                  {[
                    { value: "developer", label: "Developer / Engineer" },
                    { value: "fullstack", label: "Full-stack / Indie Hacker" },
                    { value: "devops", label: "DevOps / SRE / Platform Engineer" },
                    { value: "cto", label: "CTO / Tech Lead / Engineering Manager" },
                    { value: "founder", label: "Founder / CEO (non-technical)" },
                    { value: "student", label: "Student / Learning" },
                  ].map((opt) => (
                    <OptionButton
                      key={opt.value}
                      selected={answers.role === opt.value}
                      label={opt.label}
                      onClick={() => setAnswers({ ...answers, role: opt.value })}
                    />
                  ))}
                </div>
                <div className="space-y-2">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Team size</p>
                  {[
                    { value: "solo", label: "Just me" },
                    { value: "small", label: "2-5 people" },
                    { value: "medium", label: "6-20 people" },
                    { value: "large", label: "21-100 people" },
                    { value: "enterprise", label: "100+ people" },
                  ].map((opt) => (
                    <OptionButton
                      key={opt.value}
                      selected={answers.team_size === opt.value}
                      label={opt.label}
                      onClick={() => setAnswers({ ...answers, team_size: opt.value })}
                    />
                  ))}
                </div>
                <div className="space-y-2">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Company / project name <span className="text-neutral-600">(optional)</span></p>
                  <input
                    type="text"
                    value={answers.company_name}
                    onChange={(e) => setAnswers({ ...answers, company_name: e.target.value })}
                    placeholder="e.g. Acme Inc."
                    className="w-full rounded-lg border border-border bg-surface-100 px-4 py-2.5 text-sm text-neutral-200 placeholder-neutral-600 focus:border-accent-500 focus:outline-none"
                  />
                </div>
              </div>
            </div>
          )}

          {/* Step 3: Current Setup — provider + spend */}
          {step === 2 && (
            <div className="text-center space-y-4">
              <Server className="mx-auto h-12 w-12 text-emerald-400" />
              <h2 className="text-xl font-semibold text-white">Your current setup</h2>
              <p className="text-sm text-neutral-400">Understanding where you are helps us get you started faster.</p>
              <div className="mt-6 space-y-5 text-left">
                <div className="space-y-2">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Where do you host today?</p>
                  {[
                    { value: "nowhere", label: "Nowhere yet — this is my first deploy" },
                    { value: "heroku", label: "Heroku" },
                    { value: "vercel", label: "Vercel / Netlify" },
                    { value: "railway", label: "Railway / Render / Fly.io" },
                    { value: "aws", label: "AWS (EC2, ECS, EKS)" },
                    { value: "gcp", label: "Google Cloud (GKE, Cloud Run)" },
                    { value: "azure", label: "Azure" },
                    { value: "digitalocean", label: "DigitalOcean / Hetzner / Linode" },
                    { value: "self_hosted", label: "Self-hosted / bare metal" },
                    { value: "other", label: "Other" },
                  ].map((opt) => (
                    <OptionButton
                      key={opt.value}
                      selected={answers.current_provider === opt.value}
                      label={opt.label}
                      onClick={() => setAnswers({ ...answers, current_provider: opt.value })}
                    />
                  ))}
                </div>
                <div className="space-y-2">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Monthly hosting spend</p>
                  {[
                    { value: "0", label: "$0 — not spending yet" },
                    { value: "under_50", label: "Under $50 / month" },
                    { value: "50_200", label: "$50 – $200 / month" },
                    { value: "200_500", label: "$200 – $500 / month" },
                    { value: "500_2000", label: "$500 – $2,000 / month" },
                    { value: "over_2000", label: "$2,000+ / month" },
                  ].map((opt) => (
                    <OptionButton
                      key={opt.value}
                      selected={answers.monthly_spend === opt.value}
                      label={opt.label}
                      onClick={() => setAnswers({ ...answers, monthly_spend: opt.value })}
                    />
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Step 4: Goals — pain points, timeline, traffic */}
          {step === 3 && (
            <div className="text-center space-y-4">
              <Target className="mx-auto h-12 w-12 text-orange-400" />
              <h2 className="text-xl font-semibold text-white">What matters most?</h2>
              <p className="text-sm text-neutral-400">We&apos;ll prioritize your experience based on this.</p>
              <div className="mt-6 space-y-5 text-left">
                <div className="space-y-2">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Biggest pain point right now</p>
                  {[
                    { value: "cost", label: "Hosting costs too high" },
                    { value: "complexity", label: "Too complex to set up and maintain" },
                    { value: "scaling", label: "Hard to scale when traffic spikes" },
                    { value: "speed", label: "Deployments are too slow" },
                    { value: "lock_in", label: "Vendor lock-in concerns" },
                    { value: "compliance", label: "Need compliance / security features" },
                    { value: "support", label: "Bad support / on my own" },
                    { value: "none", label: "No pain — just exploring" },
                  ].map((opt) => (
                    <OptionButton
                      key={opt.value}
                      selected={answers.biggest_pain === opt.value}
                      label={opt.label}
                      onClick={() => setAnswers({ ...answers, biggest_pain: opt.value })}
                    />
                  ))}
                </div>
                <div className="space-y-2">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">When do you need to go live?</p>
                  {[
                    { value: "exploring", label: "Just exploring — no rush" },
                    { value: "this_week", label: "This week" },
                    { value: "this_month", label: "This month" },
                    { value: "next_quarter", label: "Next 1-3 months" },
                    { value: "already_live", label: "Already live — migrating" },
                  ].map((opt) => (
                    <OptionButton
                      key={opt.value}
                      selected={answers.timeline === opt.value}
                      label={opt.label}
                      onClick={() => setAnswers({ ...answers, timeline: opt.value })}
                    />
                  ))}
                </div>
                <div className="space-y-2">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Expected monthly traffic <span className="text-neutral-600">(optional)</span></p>
                  {[
                    { value: "starting", label: "Just starting — minimal traffic" },
                    { value: "under_10k", label: "Under 10,000 users" },
                    { value: "10k_100k", label: "10,000 – 100,000 users" },
                    { value: "100k_1m", label: "100,000 – 1 million users" },
                    { value: "over_1m", label: "1 million+ users" },
                  ].map((opt) => (
                    <OptionButton
                      key={opt.value}
                      selected={answers.expected_traffic === opt.value}
                      label={opt.label}
                      onClick={() => setAnswers({ ...answers, expected_traffic: opt.value })}
                    />
                  ))}
                </div>
                <div className="space-y-2">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Most important feature <span className="text-neutral-600">(optional)</span></p>
                  {[
                    { value: "auto_scaling", label: "Auto-scaling" },
                    { value: "managed_db", label: "Managed databases" },
                    { value: "custom_domains", label: "Custom domains & SSL" },
                    { value: "cicd", label: "Built-in CI/CD" },
                    { value: "monitoring", label: "Monitoring & alerts" },
                    { value: "security", label: "Security & compliance" },
                    { value: "cost_control", label: "Cost control & transparency" },
                    { value: "team_collab", label: "Team collaboration & RBAC" },
                  ].map((opt) => (
                    <OptionButton
                      key={opt.value}
                      selected={answers.most_important === opt.value}
                      label={opt.label}
                      onClick={() => setAnswers({ ...answers, most_important: opt.value })}
                    />
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Step 5: Tech stack (multi-select) */}
          {step === 4 && (
            <div className="text-center space-y-4">
              <Code className="mx-auto h-12 w-12 text-purple-400" />
              <h2 className="text-xl font-semibold text-white">What&apos;s your stack?</h2>
              <p className="text-sm text-neutral-400">Select all that apply — we&apos;ll suggest relevant templates and guides.</p>
              <div className="mt-6 space-y-3 text-left">
                <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Languages & Frameworks</p>
                <div className="flex flex-wrap gap-2">
                  {[
                    "Node.js", "Python", "Go", "Rust", "Java", "PHP",
                    "Next.js", "React", "Vue", "Django", "Rails", "Laravel",
                    ".NET", "Elixir",
                  ].map((tech) => (
                    <MultiOptionButton
                      key={tech}
                      selected={answers.stack.includes(tech)}
                      label={tech}
                      onClick={() => toggleStack(tech)}
                    />
                  ))}
                </div>
                <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide mt-4">Databases & Infrastructure</p>
                <div className="flex flex-wrap gap-2">
                  {[
                    "PostgreSQL", "MySQL", "MongoDB", "Redis",
                    "Docker", "Kubernetes", "Terraform",
                  ].map((tech) => (
                    <MultiOptionButton
                      key={tech}
                      selected={answers.stack.includes(tech)}
                      label={tech}
                      onClick={() => toggleStack(tech)}
                    />
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Step 6: How did you hear about us? */}
          {step === 5 && (
            <div className="text-center space-y-4">
              <Megaphone className="mx-auto h-12 w-12 text-amber-400" />
              <h2 className="text-xl font-semibold text-white">How did you find Zenith?</h2>
              <p className="text-sm text-neutral-400">Help us understand where to reach more people like you.</p>
              <div className="mt-6 space-y-2 text-left">
                <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Select one</p>
                {[
                  { value: "google", label: "Google / Search engine" },
                  { value: "youtube", label: "YouTube" },
                  { value: "twitter", label: "Twitter / X" },
                  { value: "linkedin", label: "LinkedIn" },
                  { value: "reddit", label: "Reddit / Hacker News" },
                  { value: "friend", label: "Friend / Colleague recommended" },
                  { value: "blog", label: "Blog post / Article" },
                  { value: "github", label: "GitHub" },
                  { value: "producthunt", label: "Product Hunt" },
                  { value: "conference", label: "Conference / Meetup" },
                  { value: "other", label: "Other" },
                ].map((opt) => (
                  <OptionButton
                    key={opt.value}
                    selected={answers.discovery === opt.value}
                    label={opt.label}
                    onClick={() => setAnswers({ ...answers, discovery: opt.value })}
                  />
                ))}
              </div>
            </div>
          )}

          {/* Step 7: Ready! */}
          {step === 6 && (
            <div className="text-center space-y-4">
              <PartyPopper className="mx-auto h-12 w-12 text-amber-400" />
              <h2 className="text-xl font-semibold text-white">You&apos;re All Set!</h2>
              <p className="text-sm text-neutral-400">Thanks for telling us about yourself. Your workspace is ready!</p>
              <div className="mt-6">
                <button
                  onClick={finish}
                  className="w-full rounded-lg bg-accent-500 px-6 py-3 text-sm font-semibold text-white hover:bg-accent-600 transition-colors"
                >
                  Go to Dashboard
                </button>
              </div>
              <div className="mt-2 grid grid-cols-2 gap-3">
                <button
                  onClick={async () => { await finish(); router.push("/apps"); }}
                  className="rounded-lg border border-border bg-surface-100 p-3 text-center text-sm text-neutral-400 hover:border-accent-500/40 hover:text-neutral-300 transition-colors"
                >
                  <Code className="mx-auto mb-1 h-4 w-4 text-blue-400" />
                  Deploy an App
                </button>
                <button
                  onClick={async () => { await finish(); router.push("/databases"); }}
                  className="rounded-lg border border-border bg-surface-100 p-3 text-center text-sm text-neutral-400 hover:border-accent-500/40 hover:text-neutral-300 transition-colors"
                >
                  <svg className="mx-auto mb-1 h-4 w-4 text-purple-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5V19A9 3 0 0 0 21 19V5"/><path d="M3 12A9 3 0 0 0 21 12"/></svg>
                  Add Database
                </button>
              </div>
              <div className="mt-3 rounded-lg border border-accent-500/20 bg-accent-500/5 px-4 py-3">
                <p className="text-sm text-accent-400 font-medium">Share Zenith, get 1 month Pro free</p>
                <button onClick={() => router.push("/settings?tab=referral")} className="text-xs text-accent-300 hover:text-accent-200 mt-1 inline-block">
                  Get your referral link
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Footer — no skip, Continue disabled until selection made */}
        {step < steps.length - 1 && (
          <div className="flex items-center justify-between border-t border-border px-6 py-4">
            <button
              onClick={prev}
              disabled={step === 0}
              className="flex items-center gap-1 text-sm text-neutral-500 hover:text-white disabled:invisible"
            >
              <ChevronLeft className="h-4 w-4" /> Back
            </button>
            <button
              onClick={next}
              disabled={!canProceed()}
              className="rounded-lg bg-accent-500 px-6 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Continue
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

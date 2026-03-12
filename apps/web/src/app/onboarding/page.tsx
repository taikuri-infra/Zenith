"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Rocket, User, Megaphone, Code, PartyPopper, ChevronLeft } from "lucide-react";
import { getApi } from "@/lib/get-api";
import Link from "next/link";

const steps = [
  { title: "Welcome", icon: Rocket },
  { title: "About You", icon: User },
  { title: "Discovery", icon: Megaphone },
  { title: "Stack", icon: Code },
  { title: "Ready", icon: PartyPopper },
];

type Answers = {
  use_case: string;
  role: string;
  team_size: string;
  discovery: string;
  stack: string[];
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
    discovery: "",
    stack: [],
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
      await onboarding.update(s, completed);
    } catch {
      // non-blocking
    }
  };

  const canProceed = (): boolean => {
    switch (step) {
      case 0: return answers.use_case !== "";
      case 1: return answers.role !== "" && answers.team_size !== "";
      case 2: return answers.discovery !== "";
      case 3: return answers.stack.length > 0;
      case 4: return true;
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
          <div className="flex items-center gap-2">
            {steps.map((s, i) => (
              <div key={i} className="flex items-center gap-2">
                <div
                  className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-bold transition-colors ${
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
                  <div className={`h-px w-4 ${i < step ? "bg-accent-500/40" : "bg-surface-200"}`} />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="px-6 py-8">
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
                  { value: "learn", label: "Learn cloud-native and Kubernetes" },
                  { value: "migrate", label: "Migrate from Heroku / Vercel / Railway" },
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

          {/* Step 2: About you — role + team size */}
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
                    { value: "large", label: "20+ people" },
                  ].map((opt) => (
                    <OptionButton
                      key={opt.value}
                      selected={answers.team_size === opt.value}
                      label={opt.label}
                      onClick={() => setAnswers({ ...answers, team_size: opt.value })}
                    />
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Step 3: How did you hear about us? */}
          {step === 2 && (
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

          {/* Step 4: Tech stack (multi-select) */}
          {step === 3 && (
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

          {/* Step 5: Ready! */}
          {step === 4 && (
            <div className="text-center space-y-4">
              <PartyPopper className="mx-auto h-12 w-12 text-amber-400" />
              <h2 className="text-xl font-semibold text-white">You&apos;re All Set!</h2>
              <p className="text-sm text-neutral-400">Your workspace is ready. Deploy your first app now.</p>
              <div className="mt-6 grid grid-cols-2 gap-3">
                <Link
                  href="/apps"
                  onClick={() => trackStep(steps.length, true)}
                  className="rounded-lg border border-border bg-surface-100 p-4 text-center text-sm text-neutral-300 hover:border-accent-500/40 transition-colors"
                >
                  <Code className="mx-auto mb-1.5 h-5 w-5 text-blue-400" />
                  Deploy App
                </Link>
                <Link
                  href="/databases"
                  onClick={() => trackStep(steps.length, true)}
                  className="rounded-lg border border-border bg-surface-100 p-4 text-center text-sm text-neutral-300 hover:border-accent-500/40 transition-colors"
                >
                  <svg className="mx-auto mb-1.5 h-5 w-5 text-purple-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5V19A9 3 0 0 0 21 19V5"/><path d="M3 12A9 3 0 0 0 21 12"/></svg>
                  Add Database
                </Link>
              </div>
              <div className="mt-4 rounded-lg border border-accent-500/20 bg-accent-500/5 px-4 py-3">
                <p className="text-sm text-accent-400 font-medium">Share Zenith, get 1 month Pro free</p>
                <Link href="/settings?tab=referral" className="text-xs text-accent-300 hover:text-accent-200 mt-1 inline-block">
                  Get your referral link
                </Link>
              </div>
            </div>
          )}
        </div>

        {/* Footer — no skip, Continue disabled until selection made */}
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
            {step === steps.length - 1 ? "Go to Dashboard" : "Continue"}
          </button>
        </div>
      </div>
    </div>
  );
}

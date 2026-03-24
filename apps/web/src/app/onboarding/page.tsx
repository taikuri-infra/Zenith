"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Rocket, PartyPopper, ChevronRight, Code, Database } from "lucide-react";
import { getApi } from "@/lib/get-api";

const USE_CASES = [
  { value: "side_project", label: "Ship my side project / portfolio" },
  { value: "saas", label: "Build and launch a SaaS product" },
  { value: "startup", label: "Deploy for my startup / company" },
  { value: "agency", label: "Host client projects (agency / freelancer)" },
  { value: "learn", label: "Learn cloud-native and Kubernetes" },
  { value: "migrate", label: "Migrate from another platform" },
  { value: "evaluate", label: "Evaluating PaaS options for my team" },
];

export default function OnboardingPage() {
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [userName, setUserName] = useState("");
  const [useCase, setUseCase] = useState("");
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

  const finish = async (destination = "/") => {
    try {
      await onboarding.update(2, true, useCase ? { use_case: useCase } as Record<string, unknown> : {});
    } catch {
      // non-blocking
    }
    router.push(destination);
  };

  return (
    <div className="min-h-screen bg-neutral-950 flex items-center justify-center p-4">
      <div className="w-full max-w-lg rounded-xl border border-border bg-surface-50 shadow-2xl">

        {/* Progress dots */}
        <div className="flex items-center justify-center gap-2 border-b border-border px-6 py-4">
          {[0, 1].map((i) => (
            <div
              key={i}
              className={`h-2 rounded-full transition-all ${
                i === step ? "w-6 bg-accent-500" : i < step ? "w-2 bg-accent-500/40" : "w-2 bg-surface-200"
              }`}
            />
          ))}
        </div>

        {/* Content */}
        <div className="px-6 py-8">
          {step === 0 && (
            <div className="space-y-6">
              <div className="text-center space-y-2">
                <Rocket className="mx-auto h-12 w-12 text-accent-400" />
                <h2 className="text-xl font-semibold text-white">
                  Welcome to Zenith{userName ? `, ${userName}` : ""}!
                </h2>
                <p className="text-sm text-neutral-400">
                  Your workspace is ready. What are you building?{" "}
                  <span className="text-neutral-600">(optional)</span>
                </p>
              </div>
              <div className="space-y-2">
                {USE_CASES.map((opt) => (
                  <button
                    key={opt.value}
                    onClick={() => setUseCase(opt.value)}
                    className={`w-full rounded-lg border px-4 py-2.5 text-left text-sm transition-colors ${
                      useCase === opt.value
                        ? "border-accent-500 bg-accent-500/10 text-accent-400"
                        : "border-border bg-surface-100 text-neutral-300 hover:border-neutral-600"
                    }`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
          )}

          {step === 1 && (
            <div className="space-y-6">
              <div className="text-center space-y-2">
                <PartyPopper className="mx-auto h-12 w-12 text-amber-400" />
                <h2 className="text-xl font-semibold text-white">You&apos;re all set!</h2>
                <p className="text-sm text-neutral-400">
                  Start by creating a project and deploying your first app.
                </p>
              </div>

              {/* Primary action */}
              <button
                onClick={() => finish("/projects/new")}
                className="flex items-center justify-between w-full rounded-lg bg-accent-500 px-4 py-3 text-left text-sm font-medium text-white hover:bg-accent-600 transition-colors"
              >
                <div>
                  <p className="font-semibold">Deploy your first app</p>
                  <p className="text-xs text-accent-200 mt-0.5">Paste a docker-compose.yml to get started</p>
                </div>
                <ChevronRight className="h-4 w-4 shrink-0" />
              </button>

              {/* Secondary actions */}
              <div className="grid grid-cols-2 gap-3">
                <button
                  onClick={() => finish("/databases")}
                  className="rounded-lg border border-border bg-surface-100 p-3 text-center text-sm text-neutral-400 hover:border-accent-500/40 hover:text-neutral-300 transition-colors"
                >
                  <Database className="mx-auto mb-1 h-4 w-4 text-purple-400" />
                  Add a database
                </button>
                <button
                  onClick={() => finish("/")}
                  className="rounded-lg border border-border bg-surface-100 p-3 text-center text-sm text-neutral-400 hover:border-accent-500/40 hover:text-neutral-300 transition-colors"
                >
                  <Code className="mx-auto mb-1 h-4 w-4 text-blue-400" />
                  Explore dashboard
                </button>
              </div>

              {/* Optional survey */}
              <div className="rounded-lg border border-neutral-800 bg-surface-100 px-4 py-3 text-center">
                <p className="text-xs text-neutral-500">
                  Help us improve —{" "}
                  <button
                    onClick={() => finish("/settings?survey=1")}
                    className="text-accent-400 hover:text-accent-300 underline underline-offset-2"
                  >
                    take a 2-min survey
                  </button>
                  {" "}and get <span className="text-white font-medium">1 month Pro free</span>.
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-border px-6 py-4">
          <button
            onClick={() => finish("/")}
            className="text-sm text-neutral-600 hover:text-neutral-400 transition-colors"
          >
            Skip
          </button>
          {step === 0 && (
            <button
              onClick={() => setStep(1)}
              className="rounded-lg bg-accent-500 px-6 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
            >
              Continue
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

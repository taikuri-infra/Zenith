"use client";

import { useState } from "react";
import { X, Rocket, Globe, Database, PartyPopper, ChevronRight, ChevronLeft } from "lucide-react";
import { getApi } from "@/lib/get-api";
import Link from "next/link";

interface OnboardingWizardProps {
  userName: string;
  onComplete: () => void;
  onDismiss: () => void;
}

const steps = [
  { title: "Welcome", icon: Rocket },
  { title: "Deploy", icon: Globe },
  { title: "Connect", icon: Database },
  { title: "Done", icon: PartyPopper },
];

export function OnboardingWizard({ userName, onComplete, onDismiss }: OnboardingWizardProps) {
  const [step, setStep] = useState(0);
  const [useCase, setUseCase] = useState("");
  const { onboarding } = getApi();

  const trackStep = async (s: number, completed = false) => {
    try {
      await onboarding.update(s, completed);
    } catch {
      // non-blocking
    }
  };

  const next = async () => {
    const nextStep = step + 1;
    if (nextStep >= steps.length) {
      await trackStep(steps.length, true);
      onComplete();
      return;
    }
    await trackStep(nextStep);
    setStep(nextStep);
  };

  const prev = () => {
    if (step > 0) setStep(step - 1);
  };

  const dismiss = async () => {
    await trackStep(step, true);
    onDismiss();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70">
      <div className="w-full max-w-lg rounded-xl border border-border bg-surface-50 shadow-2xl" onClick={(e) => e.stopPropagation()}>
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-6 py-4">
          <div className="flex items-center gap-3">
            {steps.map((s, i) => (
              <div
                key={i}
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
            ))}
          </div>
          <button onClick={dismiss} className="text-neutral-500 hover:text-white">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Content */}
        <div className="px-6 py-8">
          {step === 0 && (
            <div className="text-center space-y-4">
              <Rocket className="mx-auto h-12 w-12 text-accent-400" />
              <h2 className="text-xl font-semibold text-white">Welcome to Zenith, {userName}!</h2>
              <p className="text-sm text-neutral-400">Let&apos;s get your first app deployed in minutes.</p>
              <div className="mt-6 space-y-2 text-left">
                <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">What brings you here?</p>
                {[
                  { value: "side_project", label: "Deploy my side project" },
                  { value: "saas", label: "Build a SaaS" },
                  { value: "learn", label: "Learn cloud-native" },
                  { value: "migrate", label: "Migrate from Heroku/Vercel" },
                ].map((opt) => (
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
            <div className="text-center space-y-4">
              <Globe className="mx-auto h-12 w-12 text-blue-400" />
              <h2 className="text-xl font-semibold text-white">Deploy Your First App</h2>
              <p className="text-sm text-neutral-400">Choose how you&apos;d like to get started.</p>
              <div className="mt-6 space-y-3">
                <Link
                  href="/apps"
                  onClick={() => trackStep(2)}
                  className="flex items-center justify-between w-full rounded-lg border border-border bg-surface-100 px-4 py-3 text-left text-sm text-white hover:border-accent-500/40 transition-colors"
                >
                  <div>
                    <p className="font-medium">Deploy from Docker image</p>
                    <p className="text-xs text-neutral-500 mt-0.5">Use any Docker image from Docker Hub or a registry</p>
                  </div>
                  <ChevronRight className="h-4 w-4 text-neutral-500" />
                </Link>
                <Link
                  href="/apps"
                  onClick={() => trackStep(2)}
                  className="flex items-center justify-between w-full rounded-lg border border-border bg-surface-100 px-4 py-3 text-left text-sm text-white hover:border-accent-500/40 transition-colors"
                >
                  <div>
                    <p className="font-medium">Deploy from Git repository</p>
                    <p className="text-xs text-neutral-500 mt-0.5">Connect your GitHub repo for automatic deployments</p>
                  </div>
                  <ChevronRight className="h-4 w-4 text-neutral-500" />
                </Link>
              </div>
              <button onClick={next} className="mt-4 text-xs text-neutral-500 hover:text-neutral-300">
                Skip for now
              </button>
            </div>
          )}

          {step === 2 && (
            <div className="text-center space-y-4">
              <Database className="mx-auto h-12 w-12 text-purple-400" />
              <h2 className="text-xl font-semibold text-white">Connect a Domain</h2>
              <p className="text-sm text-neutral-400">Your app is available at a free Zenith subdomain. Upgrade to Pro for custom domains.</p>
              <div className="mt-4 rounded-lg border border-border bg-surface-100 px-4 py-3">
                <p className="text-xs text-neutral-500">Your app URL</p>
                <p className="text-sm font-mono text-accent-400 mt-1">your-app.apps.your-domain.com</p>
              </div>
              <Link
                href="/billing"
                className="inline-block mt-2 text-sm text-accent-400 hover:text-accent-300"
              >
                Upgrade to Pro for custom domains
              </Link>
            </div>
          )}

          {step === 3 && (
            <div className="text-center space-y-4">
              <PartyPopper className="mx-auto h-12 w-12 text-amber-400" />
              <h2 className="text-xl font-semibold text-white">You&apos;re All Set!</h2>
              <p className="text-sm text-neutral-400">Your Zenith workspace is ready. Here are some quick actions:</p>
              <div className="mt-6 grid grid-cols-2 gap-3">
                <Link href="/apps" className="rounded-lg border border-border bg-surface-100 p-3 text-center text-sm text-neutral-300 hover:border-accent-500/40 transition-colors">
                  <Globe className="mx-auto mb-1 h-5 w-5 text-blue-400" />
                  Deploy App
                </Link>
                <Link href="/databases" className="rounded-lg border border-border bg-surface-100 p-3 text-center text-sm text-neutral-300 hover:border-accent-500/40 transition-colors">
                  <Database className="mx-auto mb-1 h-5 w-5 text-purple-400" />
                  Add Database
                </Link>
              </div>
              <div className="mt-4 rounded-lg border border-accent-500/20 bg-accent-500/5 px-4 py-3">
                <p className="text-sm text-accent-400 font-medium">Share Zenith, get 1 month Pro free</p>
                <Link href="/settings" className="text-xs text-accent-300 hover:text-accent-200 mt-1 inline-block">
                  Get your referral link
                </Link>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
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
            className="rounded-lg bg-accent-500 px-6 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
          >
            {step === steps.length - 1 ? "Get Started" : "Continue"}
          </button>
        </div>
      </div>
    </div>
  );
}

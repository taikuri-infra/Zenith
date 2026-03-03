"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Loader2 } from "lucide-react";
import { auth } from "@/lib/api";

function AuthCallbackInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const code = searchParams.get("code");
    const errorParam = searchParams.get("error");

    if (errorParam) {
      setError(errorParam === "oauth_failed"
        ? "Authentication failed. Please try again."
        : errorParam === "invalid_state"
        ? "Invalid session. Please try again."
        : errorParam);
      return;
    }

    if (!code) {
      setError("Missing authorization code.");
      return;
    }

    auth.exchangeOAuthCode({ code })
      .then(() => {
        router.replace("/");
      })
      .catch(() => {
        setError("Failed to complete sign in. The code may have expired.");
      });
  }, [searchParams, router]);

  if (error) {
    return (
      <div className="w-full max-w-md">
        <div className="bg-neutral-900 border border-neutral-800 rounded-2xl p-8 text-center">
          <div className="w-16 h-16 bg-red-500/10 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <h2 className="text-xl font-semibold text-white mb-2">Sign in failed</h2>
          <p className="text-neutral-400 text-sm mb-6">{error}</p>
          <a
            href="/login"
            className="inline-flex items-center justify-center px-4 py-2.5 bg-emerald-600 hover:bg-emerald-500 rounded-lg text-white font-medium transition-colors"
          >
            Back to sign in
          </a>
        </div>
      </div>
    );
  }

  return (
    <div className="text-center">
      <Loader2 className="w-8 h-8 text-emerald-500 animate-spin mx-auto mb-4" />
      <p className="text-neutral-400 text-sm">Completing sign in...</p>
    </div>
  );
}

export default function AuthCallbackPage() {
  return (
    <div className="min-h-screen bg-neutral-950 flex items-center justify-center p-4">
      <Suspense fallback={
        <div className="text-center">
          <Loader2 className="w-8 h-8 text-emerald-500 animate-spin mx-auto mb-4" />
          <p className="text-neutral-400 text-sm">Loading...</p>
        </div>
      }>
        <AuthCallbackInner />
      </Suspense>
    </div>
  );
}

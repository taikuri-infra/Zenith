"use client";

import { useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense } from "react";

function RegisterRedirect() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const params = new URLSearchParams();
    params.set("mode", "register");
    // Forward UTM and referral params
    for (const key of ["utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term", "ref"]) {
      const val = searchParams.get(key);
      if (val) params.set(key, val);
    }
    router.replace(`/login?${params.toString()}`);
  }, [router, searchParams]);

  return null;
}

export default function RegisterPage() {
  return (
    <Suspense>
      <RegisterRedirect />
    </Suspense>
  );
}

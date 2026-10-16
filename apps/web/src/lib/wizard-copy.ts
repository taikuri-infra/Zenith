import { SHOW_PLANS } from "@/lib/edition";

/** Plan-dependent text in the deploy wizard. Self-host has no plans, so every line is neutral there. */
export interface WizardPlanCopy {
  /** Hint under the image source picker; null renders nothing. */
  registryHint: string | null;
  /** Body of the S3 tile shown when buckets cannot be attached in the wizard. */
  s3Hint: string;
  /** Whether the S3 tile carries the plan crown. */
  s3Crown: boolean;
  /** Line under the resources tile; null renders nothing. */
  resourcesNote: string | null;
  /** Upgrade nudge after resourcesNote; null renders nothing. */
  resourcesUpgrade: string | null;
  /** Plan name after the resources in the review step; null renders nothing. */
  reviewPlanLabel: string | null;
}

export function wizardPlanCopy(isPro: boolean): WizardPlanCopy {
  if (!SHOW_PLANS) {
    return {
      registryHint: null,
      s3Hint: "Buckets are attached from the Storage page. Access via any S3 SDK — credentials are injected automatically.",
      s3Crown: false,
      resourcesNote: null,
      resourcesUpgrade: null,
      reviewPlanLabel: null,
    };
  }
  return {
    registryHint: isPro ? null : "Upgrade to Pro for Zenith's built-in registry with image browsing",
    s3Hint: "Upgrade to Pro to attach S3 buckets for file uploads, images, and static assets. Access via any S3 SDK — credentials are injected automatically.",
    s3Crown: true,
    resourcesNote: `${isPro ? "Pro plan" : "Free plan"} per-app limit.`,
    resourcesUpgrade: isPro ? null : "Upgrade to Pro for more resources.",
    reviewPlanLabel: `(${isPro ? "Pro" : "Free"} plan)`,
  };
}

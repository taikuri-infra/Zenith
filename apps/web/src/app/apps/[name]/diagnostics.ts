import type { PodStatus } from "@/lib/api";

/** One-line hint for a pod that failed to start, shown in the app page banner. */
export function getDiagnosticHint(pod: PodStatus): string {
  const reason = pod.status_reason || "";
  const message = pod.status_message || "";
  if (reason === "CrashLoopBackOff") {
    return "Your app is crashing on startup. Check the logs for details.";
  }
  if (reason === "ErrImagePull" || reason === "ImagePullBackOff") {
    return "Could not pull the image. Verify the image was built successfully.";
  }
  if (reason === "OOMKilled" || pod.last_exit_code === 137) {
    return "Your app ran out of memory. Consider optimizing memory usage or upgrading your plan.";
  }
  if (message.includes("exec format error")) {
    return "Image was built for the wrong CPU architecture. Rebuild with --platform linux/amd64.";
  }
  return "Your app failed to start. Check the logs for more details.";
}

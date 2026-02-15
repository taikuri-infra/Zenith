"use client";

import { useState } from "react";
import { isDemoMode } from "@/lib/get-api";

interface DemoButtonProps {
  /** The button text */
  children: React.ReactNode;
  /** Original onClick handler (only called outside demo mode) */
  onClick?: () => void;
  /** Whether the button should be disabled even outside demo mode */
  disabled?: boolean;
  /** Extra CSS classes */
  className?: string;
}

/**
 * A button wrapper that, in demo mode, prevents the action and shows
 * a tooltip: "Available in your own installation".
 * Outside demo mode it behaves as a normal button.
 */
export function DemoButton({
  children,
  onClick,
  disabled,
  className = "",
}: DemoButtonProps) {
  const [showTooltip, setShowTooltip] = useState(false);
  const demo = isDemoMode();

  const handleClick = () => {
    if (demo) {
      setShowTooltip(true);
      setTimeout(() => setShowTooltip(false), 2000);
      return;
    }
    onClick?.();
  };

  return (
    <span className="relative inline-block">
      <button
        onClick={handleClick}
        disabled={disabled && !demo}
        className={`${className} ${demo ? "cursor-not-allowed opacity-60" : ""}`}
      >
        {children}
      </button>
      {showTooltip && (
        <span className="absolute -top-9 left-1/2 z-50 -translate-x-1/2 whitespace-nowrap rounded-md bg-surface-50 border border-border px-2.5 py-1 text-[11px] text-neutral-300 shadow-lg">
          Available in your own installation
        </span>
      )}
    </span>
  );
}

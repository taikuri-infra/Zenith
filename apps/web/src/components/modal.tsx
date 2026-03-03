"use client";
import { X } from "lucide-react";

const sizeClasses = {
  sm: "max-w-sm",
  md: "max-w-md",
  lg: "max-w-2xl",
  xl: "max-w-3xl",
} as const;

interface ModalProps {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
  size?: keyof typeof sizeClasses;
}

export function Modal({ title, onClose, children, size = "md" }: ModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={onClose}>
      <div className={`w-full ${sizeClasses[size]} max-h-[85vh] overflow-y-auto rounded-lg border border-border bg-surface-50 p-6`} onClick={e => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-base font-semibold text-white">{title}</h2>
          <button onClick={onClose} className="text-neutral-500 hover:text-white"><X className="h-4 w-4" /></button>
        </div>
        {children}
      </div>
    </div>
  );
}

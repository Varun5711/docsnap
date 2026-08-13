// Same dot+glow+label mechanics as cdex-admin-client's contests/status-badge.tsx,
// remapped to docsnap's evidence verification statuses.
import { cn } from "@/lib/utils/cn";

type EvidenceStatus = "certified" | "verified" | "tampered";

const statusConfig: Record<EvidenceStatus, {
  label: string;
  dotClass: string;
  textClass: string;
  animate?: boolean;
}> = {
  certified: {
    label: "Certified",
    dotClass: "bg-sky-400 shadow-[0_0_10px_rgba(56,189,248,0.5)]",
    textClass: "text-foreground",
  },
  verified: {
    label: "Verified",
    dotClass: "bg-green-500 shadow-[0_0_12px_rgba(34,197,94,0.8)]",
    textClass: "text-white drop-shadow-[0_0_5px_rgba(34,197,94,0.5)]",
    animate: true,
  },
  tampered: {
    label: "Tampered",
    dotClass: "bg-red-500 shadow-[0_0_12px_rgba(239,68,68,0.6)]",
    textClass: "text-red-400",
  },
};

export function EvidenceStatusBadge({ status }: { status: string }) {
  const config = statusConfig[status as EvidenceStatus] ?? statusConfig.certified;

  return (
    <div className="flex items-center gap-2.5">
      <span className="relative flex h-2.5 w-2.5">
        {config.animate && (
          <span
            className={cn(
              "absolute inline-flex h-full w-full rounded-full opacity-40 animate-pulse",
              config.dotClass
            )}
          />
        )}
        <span className={cn("relative inline-flex rounded-full h-2.5 w-2.5", config.dotClass)} />
      </span>
      <span className={cn("text-xs font-semibold", config.textClass)}>{config.label}</span>
    </div>
  );
}

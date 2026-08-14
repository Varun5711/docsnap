import { cn } from "@/lib/utils/cn";
import type { VerdictStatus } from "@/lib/api";
const statusConfig: Record<
  VerdictStatus,
  {
    label: string;
    dotClass: string;
    textClass: string;
  }
> = {
  SUPPORTED: {
    label: "Supported",
    dotClass: "bg-green-500 shadow-[0_0_12px_rgba(34,197,94,0.8)]",
    textClass: "text-green-400",
  },
  LIKELY_SUPPORTED: {
    label: "Likely supported",
    dotClass: "bg-green-400/70 shadow-[0_0_10px_rgba(74,222,128,0.5)]",
    textClass: "text-green-300",
  },
  MIXED: {
    label: "Mixed evidence",
    dotClass: "bg-amber-400 shadow-[0_0_10px_rgba(251,191,36,0.6)]",
    textClass: "text-amber-300",
  },
  UNVERIFIED: {
    label: "Unverified",
    dotClass: "bg-zinc-400",
    textClass: "text-zinc-400",
  },
  LIKELY_CONTRADICTED: {
    label: "Likely contradicted",
    dotClass: "bg-red-400/80 shadow-[0_0_10px_rgba(248,113,113,0.5)]",
    textClass: "text-red-300",
  },
  CONTRADICTED: {
    label: "Contradicted",
    dotClass: "bg-red-500 shadow-[0_0_12px_rgba(239,68,68,0.7)]",
    textClass: "text-red-400",
  },
};
export function VerdictBadge({
  status,
  size = "md",
}: {
  status: string;
  size?: "sm" | "md";
}) {
  const config =
    statusConfig[status as VerdictStatus] ?? statusConfig.UNVERIFIED;
  const dotSize = size === "sm" ? "h-2 w-2" : "h-2.5 w-2.5";
  const textSize = size === "sm" ? "text-xs" : "text-sm";
  return (
    <div className="flex items-center gap-2">
      <span
        className={cn(
          "relative inline-flex rounded-full",
          dotSize,
          config.dotClass,
        )}
      />
      <span className={cn("font-semibold", textSize, config.textClass)}>
        {config.label}
      </span>
    </div>
  );
}

import { cn } from "@/lib/utils";

export interface TireCompoundProps {
  /** Tire compound identifier */
  compound:
    | "soft"
    | "medium"
    | "hard"
    | "intermediate"
    | "wet"
    | string
    | null
    | undefined;
  /** Render as small inline badge (default) or larger display */
  size?: "sm" | "md";
  /** Optional className passthrough */
  className?: string;
}

interface TireCompoundConfig {
  letter: string;
  background: string;
  textColor: string;
  label: string;
}

const TIRE_COMPOUNDS: Record<string, TireCompoundConfig> = {
  soft: { letter: "S", background: "#e8002d", textColor: "#ffffff", label: "Soft" },
  medium: { letter: "M", background: "#ffc107", textColor: "#000000", label: "Medium" },
  hard: { letter: "H", background: "#ffffff", textColor: "#000000", label: "Hard" },
  intermediate: {
    letter: "I",
    background: "#2196f3",
    textColor: "#ffffff",
    label: "Intermediate",
  },
  wet: { letter: "W", background: "#4caf50", textColor: "#ffffff", label: "Wet" },
};

const UNKNOWN_COMPOUND: TireCompoundConfig = {
  letter: "?",
  background: "#8888aa",
  textColor: "#ffffff",
  label: "Unknown",
};

const SIZE_CLASSES: Record<NonNullable<TireCompoundProps["size"]>, string> = {
  sm: "h-6 w-6 text-[10px]",
  md: "h-8 w-8 text-xs",
};

export default function TireCompound({
  compound,
  size = "sm",
  className,
}: TireCompoundProps) {
  const config = compound
    ? (TIRE_COMPOUNDS[compound.toLowerCase()] ?? UNKNOWN_COMPOUND)
    : UNKNOWN_COMPOUND;

  return (
    <span
      data-testid="tire-compound"
      data-compound={compound ?? "unknown"}
      title={config.label}
      aria-label={config.label}
      className={cn(
        "inline-flex items-center justify-center rounded-full font-display font-bold leading-none",
        SIZE_CLASSES[size],
        className,
      )}
      style={{ backgroundColor: config.background, color: config.textColor }}
    >
      {config.letter}
    </span>
  );
}

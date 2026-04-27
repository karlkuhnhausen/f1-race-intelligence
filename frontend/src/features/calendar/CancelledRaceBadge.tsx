interface CancelledRaceBadgeProps {
  label?: string;
  reason?: string;
}

export default function CancelledRaceBadge({ label, reason }: CancelledRaceBadgeProps) {
  return (
    <span
      title={reason}
      className="cancelled ml-2 inline-flex items-center rounded-md bg-negative/20 px-2 py-0.5 text-xs font-display font-bold uppercase tracking-wider text-negative"
    >
      {label ?? 'Cancelled'}
    </span>
  );
}

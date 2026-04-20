interface CancelledRaceBadgeProps {
  label?: string;
  reason?: string;
}

export default function CancelledRaceBadge({ label, reason }: CancelledRaceBadgeProps) {
  return (
    <span
      title={reason}
      style={{
        display: 'inline-block',
        padding: '0.15em 0.5em',
        fontSize: '0.8em',
        fontWeight: 'bold',
        color: '#fff',
        backgroundColor: '#d32f2f',
        borderRadius: '4px',
        marginLeft: '0.5em',
      }}
    >
      {label ?? 'Cancelled'}
    </span>
  );
}

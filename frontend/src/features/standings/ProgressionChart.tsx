import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';

export interface ProgressionEntry {
  name: string;
  color: string;
  pointsByRound: number[];
}

export interface ProgressionChartProps {
  rounds: string[];
  entries: ProgressionEntry[];
}

export default function ProgressionChart({ rounds, entries }: ProgressionChartProps) {
  if (rounds.length <= 1) {
    return (
      <p className="text-muted-foreground text-center py-8">
        Not enough data for progression chart (need at least 2 rounds).
      </p>
    );
  }

  // Transform into recharts data format: [{round: "Round 1", "Verstappen": 25, "Norris": 18}, ...]
  const data = rounds.map((round, i) => {
    const point: Record<string, string | number> = { round };
    for (const entry of entries) {
      point[entry.name] = entry.pointsByRound[i] ?? 0;
    }
    return point;
  });

  return (
    <div data-testid="progression-chart" className="w-full rounded-lg border border-border bg-surface p-4">
      <ResponsiveContainer width="100%" height={450}>
        <LineChart data={data} margin={{ top: 10, right: 30, left: 20, bottom: 10 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#2a2a38" />
          <XAxis
            dataKey="round"
            stroke="#8888aa"
            tick={{ fontSize: 11, fill: '#ffffff' }}
          />
          <YAxis
            stroke="#8888aa"
            tick={{ fontSize: 11, fill: '#ffffff' }}
            label={{ value: 'Points', angle: -90, position: 'insideLeft', offset: -5, fill: '#ffffff', fontSize: 12 }}
          />
          <Tooltip
            cursor={{ stroke: '#555', strokeDasharray: '3 3' }}
            content={({ active, payload, label }) => {
              if (!active || !payload || payload.length === 0) return null;
              const sorted = [...payload]
                .filter((e) => e.value != null)
                .sort((a, b) => (b.value as number) - (a.value as number));
              return (
                <div style={{
                  backgroundColor: 'rgba(26, 26, 35, 0.85)',
                  border: '1px solid #2a2a38',
                  borderRadius: '6px',
                  padding: '6px 10px',
                  fontSize: '11px',
                  backdropFilter: 'blur(4px)',
                  boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
                }}>
                  <p style={{ color: '#ffffff', marginBottom: 3, fontWeight: 600 }}>{label}</p>
                  {sorted.map((entry) => (
                    <div key={String(entry.dataKey)} style={{ display: 'flex', gap: 6, lineHeight: '1.6' }}>
                      <span style={{ color: '#999999', minWidth: 32, textAlign: 'right', fontWeight: 600 }}>{entry.value}</span>
                      <span style={{ color: entry.color, fontWeight: 600 }}>{entry.name}</span>
                    </div>
                  ))}
                </div>
              );
            }}
          />
          <Legend wrapperStyle={{ fontSize: '11px', paddingTop: '16px' }} />
          {entries.map((entry) => (
            <Line
              key={entry.name}
              type="monotone"
              dataKey={entry.name}
              stroke={entry.color || '#888'}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

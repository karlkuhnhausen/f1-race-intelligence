import { useEffect, useState } from 'react';
import { fetchConstructorDriverBreakdown, type ConstructorBreakdownResponse } from './standingsApi';

interface ConstructorBreakdownProps {
  teamName: string;
  year: number;
}

export default function ConstructorBreakdown({ teamName, year }: ConstructorBreakdownProps) {
  const [data, setData] = useState<ConstructorBreakdownResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetchConstructorDriverBreakdown(year, teamName)
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [year, teamName]);

  if (loading) return <div className="py-2 px-4 text-xs text-muted-foreground">Loading…</div>;
  if (!data || data.drivers.length === 0) return null;

  return (
    <div data-testid="constructor-breakdown" className="px-4 py-2 bg-background/20">
      <table className="w-full text-xs">
        <thead>
          <tr className="text-muted-foreground">
            <th className="text-left py-1">Driver</th>
            <th className="text-left py-1">Pos</th>
            <th className="text-left py-1">Pts</th>
            <th className="text-left py-1">Wins</th>
            <th className="text-left py-1">Podiums</th>
            <th className="text-left py-1">%</th>
          </tr>
        </thead>
        <tbody>
          {data.drivers.map((d) => (
            <tr key={d.driver_number} className="text-foreground">
              <td className="py-1 font-display">{d.driver_name}</td>
              <td className="py-1 font-mono">{d.position}</td>
              <td className="py-1 font-mono">{d.points}</td>
              <td className="py-1 font-mono">{d.wins}</td>
              <td className="py-1 font-mono">{d.podiums}</td>
              <td className="py-1">
                <div className="flex items-center gap-1">
                  <div
                    className="h-2 rounded bg-accent-red/60"
                    style={{ width: `${Math.min(d.points_percentage, 100)}%`, minWidth: '2px' }}
                  />
                  <span className="font-mono">{d.points_percentage.toFixed(0)}%</span>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

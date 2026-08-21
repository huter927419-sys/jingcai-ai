import type { PlayerXI, SidePreview, TeamPreview } from "./api";

export function LineupBoard({ preview }: { preview: TeamPreview }) {
  return (
    <div className="xi-board">
      <SideCard side={preview.home} />
      <SideCard side={preview.away} />
    </div>
  );
}

function SideCard({ side }: { side: SidePreview }) {
  const rows = pitchRows(side.starters ?? []);
  return (
    <div className="xi-card">
      <div className="xi-head">
        <b>{side.name}</b>
        <em>{side.formation || "阵型未定"}</em>
        {side.avgRating > 0 ? <span className={`xi-avg ${tone(side.avgRating)}`}>{side.avgRating.toFixed(1)}</span> : null}
      </div>
      <div className="pitch">
        {rows.length ? (
          rows.map((row, i) => (
            <div className="pitch-row" key={i}>
              {row.map((p) => (
                <div className="pitch-p" key={`${p.no}-${p.name}`}>
                  <i>{p.no || "·"}</i>
                  <b>{shortName(p.name)}</b>
                </div>
              ))}
            </div>
          ))
        ) : (
          <div className="pred muted">首发还没出。</div>
        )}
      </div>
      <div className="block-h">近期评分</div>
      {side.form?.length ? (
        <div className="form-list">
          {side.form.map((g) => (
            <div className="form-row" key={`${g.date}-${g.home}-${g.away}`}>
              <span>{g.date}</span>
              <span className="form-vs">
                {g.home} {g.score} {g.away}
              </span>
              <em className={`res ${g.result}`}>{g.result}</em>
              <b className={tone(g.rating)}>{g.rating.toFixed(1)}</b>
            </div>
          ))}
        </div>
      ) : (
        <div className="pred muted">近期战绩还没齐。</div>
      )}
    </div>
  );
}

function pitchRows(starters: PlayerXI[]): PlayerXI[][] {
  const gk = starters.filter((p) => p.pos === "守门员");
  const d = starters.filter((p) => p.pos === "后卫");
  const m = starters.filter((p) => p.pos === "中场");
  const f = starters.filter((p) => p.pos === "前锋");
  const rest = starters.filter((p) => !["守门员", "后卫", "中场", "前锋"].includes(p.pos || ""));
  return [f, m.concat(rest), d, gk].filter((r) => r.length);
}

function shortName(name: string): string {
  const s = name.trim();
  if (s.length <= 5) return s;
  return s.slice(-5);
}

function tone(v: number): string {
  if (v >= 7) return "hi";
  if (v >= 5) return "mid";
  return "lo";
}

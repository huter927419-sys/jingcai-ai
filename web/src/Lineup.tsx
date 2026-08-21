import type { PlayerXI, SidePreview, TeamPreview } from "./api";

export function LineupBoard({ preview }: { preview: TeamPreview }) {
  return (
    <div className="xi-board">
      <SideCard side={preview.home} team="home" />
      <SideCard side={preview.away} team="away" />
    </div>
  );
}

function SideCard({ side, team }: { side: SidePreview; team: "home" | "away" }) {
  const rows = formationRows(side.starters ?? [], side.formation);
  const positions = rowPositions(rows.length);
  return (
    <div className={`xi-card ${team}`}>
      <div className="xi-head">
        <b>{side.name}</b>
        <em>{side.formation || "阵型未定"}</em>
        {side.avgRating > 0 ? <span className={`xi-avg ${tone(side.avgRating)}`}>{side.avgRating.toFixed(1)}</span> : null}
      </div>
      <div className="squad-status">
        <span>首发 <b>{side.starters?.length || 0}/11</b></span>
        <span>替补 <b>{side.bench?.length || 0}</b></span>
        <span className="unconfirmed">伤停待确认</span>
      </div>
      <div className="pitch" aria-label={`${side.name} ${side.formation || "阵型未定"} 战术站位`}>
        <div className="pitch-markings" aria-hidden>
          <i className="pitch-half" />
          <i className="pitch-circle" />
          <i className="pitch-box top" />
          <i className="pitch-box bottom" />
          <i className="pitch-goal top" />
          <i className="pitch-goal bottom" />
        </div>
        {rows.length ? (
          <div className="formation-lines">
            {rows.map((row, i) => (
            <div className={`pitch-row line-${i + 1}`} style={{ top: `${positions[i]}%` }} key={i}>
              {row.map((p) => (
                <div className="pitch-p" key={`${p.no}-${p.name}`}>
                  <i>{p.no || "·"}</i>
                  <b>{shortName(p.name)}</b>
                  <small>{positionLabel(p.pos)}</small>
                </div>
              ))}
            </div>
            ))}
          </div>
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

function formationRows(starters: PlayerXI[], formation: string): PlayerXI[][] {
  const gk = starters.filter((p) => p.pos === "守门员");
  const d = starters.filter((p) => p.pos === "后卫");
  const m = starters.filter((p) => p.pos === "中场");
  const f = starters.filter((p) => p.pos === "前锋");
  const rest = starters.filter((p) => !["守门员", "后卫", "中场", "前锋"].includes(p.pos || ""));
  const midfield = m.concat(rest);
  const parsed = formation.split("-").map(Number).filter((n) => Number.isFinite(n) && n > 0);
  const rows: PlayerXI[][] = [];
  if (f.length) rows.push(...splitLine(f, parsed.at(-1) || f.length));
  if (midfield.length) rows.push(...splitLine(midfield, midfield.length > 4 ? Math.ceil(midfield.length / 2) : midfield.length));
  if (d.length) rows.push(...splitLine(d, parsed[0] || d.length));
  if (gk.length) rows.push(gk);
  return rows;
}

function splitLine(players: PlayerXI[], preferred: number): PlayerXI[][] {
  if (players.length <= 4) return [players];
  const firstSize = Math.min(4, Math.max(2, preferred > 4 ? Math.ceil(players.length / 2) : preferred));
  return [players.slice(0, firstSize), players.slice(firstSize)].filter((row) => row.length);
}

function rowPositions(count: number): number[] {
  if (count === 5) return [19, 34, 61, 76, 90];
  if (count === 4) return [19, 37, 67, 90];
  if (count === 3) return [23, 58, 88];
  if (count === 2) return [30, 85];
  return [50];
}

function shortName(name: string): string {
  const s = name.trim();
  if (s.length <= 5) return s;
  return s.slice(-5);
}

function positionLabel(pos?: string): string {
  if (pos === "守门员") return "GK";
  if (pos === "后卫") return "DEF";
  if (pos === "中场") return "MID";
  if (pos === "前锋") return "FWD";
  return "XI";
}

function tone(v: number): string {
  if (v >= 7) return "hi";
  if (v >= 5) return "mid";
  return "lo";
}

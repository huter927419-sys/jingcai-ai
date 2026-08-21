import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { fetchToday, fmtKick, fmtPrice, playRows, sortMatches, type MatchRow, type MatchSort } from "../api";
import Layout from "../Layout";
import { ScanShape } from "../Charts";
import { IconBall, IconClock } from "../Icons";

export default function Today() {
  const [rows, setRows] = useState<ReturnType<typeof sortMatches>>([]);
  const [finished, setFinished] = useState(0);
  const [sort, setSort] = useState<MatchSort>("num");
  const [err, setErr] = useState("");

  async function load() {
    try {
      setErr("");
      const j = await fetchToday();
      setRows(j.matches.filter((m) => !m.finished && m.status !== "完场"));
      setFinished(j.finished);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "加载失败");
    }
  }

  useEffect(() => {
    void load();
    const t = setInterval(() => void load(), 20000);
    return () => clearInterval(t);
  }, []);

  const live = rows.filter((m) => m.status === "临场").length;

  return (
    <Layout>
      <div className="page-head">
        <div>
          <h1>今日工作台</h1>
          <div className="sub">
            <IconBall size={14} />
            {rows.length} 场 · {live ? `${live} 场临场 · ` : ""}
            {finished ? `${finished} 场完场已移到结果 · ` : ""}
            先扫走势，再进场看票面和价值
          </div>
        </div>
        <div className="switch">
          <button className={sort === "num" ? "btn" : "btn ghost"} onClick={() => setSort("num")}>
            序号
          </button>
          <button className={sort === "kick" ? "btn" : "btn ghost"} onClick={() => setSort("kick")}>
            开赛时间
          </button>
        </div>
      </div>
      {err ? <div className="err">{err}。先启动后端：make backend</div> : null}
      {rows.length === 0 ? (
        <div className="empty">
          {finished > 0 ? (
            <>
              今日场次已完场。
              <Link to="/results">去结果页看专家评比</Link>
            </>
          ) : (
            "还没有场次。"
          )}
        </div>
      ) : (
        <div className="list">
          {sortMatches(rows, sort).map((m) => (
            <ScanCard key={m.id} m={m} />
          ))}
        </div>
      )}
    </Layout>
  );
}

function ScanCard({ m }: { m: MatchRow }) {
  const plays = playRows(m.odds);
  const std = plays.find((p) => p.key === "std");
  const hc = plays.find((p) => p.key === "hc");
  const ou = plays.find((p) => p.key === "ou");
  const shape = m.shape;

  return (
    <Link className="match-card scan-card" to={`/matches/${m.id}`}>
      <div className="match-card-head">
        <div>
          <div className="match-id">
            <span className="num">{m.numStr}</span>
            <span className="league">{m.leagueAbb || m.league}</span>
          </div>
          <div className="teams">
            {m.home} vs {m.away}
          </div>
          <div className="kick">
            <IconClock size={12} />
            {fmtKick(m.kickoff)}
          </div>
        </div>
        <div className={`pill ${m.status === "临场" || m.status === "完场" ? "on" : ""}`}>{m.status}</div>
      </div>
      <div className="scan-body">
        {shape ? (
          <ScanShape home={shape.homeWin} draw={shape.draw} away={shape.awayWin} over={shape.over25} />
        ) : (
          <div className="pred muted">走势还在分析。</div>
        )}
        <div className="scan-ticket">
          <div className="block-h">竞彩票面</div>
          <div className="scan-prices">
            <div>
              <span>胜</span>
              <b>{fmtPrice(std?.h)}</b>
            </div>
            <div>
              <span>平</span>
              <b>{fmtPrice(std?.d)}</b>
            </div>
            <div>
              <span>负</span>
              <b>{fmtPrice(std?.a)}</b>
            </div>
          </div>
          {std && (std.pH || std.pD || std.pA) ? (
            <div className="play-bar">
              {std.pH ? <i className="h live" style={{ width: `${std.pH}%` }} /> : null}
              {std.pD ? <i className="d live" style={{ width: `${std.pD}%` }} /> : null}
              {std.pA ? <i className="a live" style={{ width: `${std.pA}%` }} /> : null}
            </div>
          ) : null}
          <div className="scan-meta">
            <span>{hc?.empty ? "让球 —" : hc?.label}</span>
            <span>
              {ou?.empty
                ? "大小 —"
                : `大 ${fmtPrice(ou?.h, true)} / 小 ${fmtPrice(ou?.a, true)}`}
            </span>
          </div>
        </div>
      </div>
    </Link>
  );
}

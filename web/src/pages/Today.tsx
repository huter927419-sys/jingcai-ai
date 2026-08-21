import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { fetchToday, fmtKick, fmtPrice, playRows, sortMatches, type MatchRow, type MatchSort } from "../api";
import Layout from "../Layout";
import { ScanShape } from "../Charts";
import { IconBall, IconClock, IconPulse, IconSpark } from "../Icons";

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
  const analyzed = rows.filter((m) => m.shape).length;

  return (
    <Layout>
      <div className="page-head today-head">
        <div>
          <div className="eyebrow"><span /> LIVE MATCH INTELLIGENCE</div>
          <h1>今日赛事研判</h1>
          <div className="sub">
            基于赛程、市场赔率与多模型共识生成的实时决策视图
          </div>
        </div>
        <div className="refresh-note"><i /><span>数据自动更新<small>每 20 秒同步一次</small></span></div>
      </div>
      <div className="overview-grid">
        <div className="overview-item"><span>今日待赛</span><strong>{rows.length}</strong><small>场赛事</small><IconBall size={18} /></div>
        <div className="overview-item"><span>临场监测</span><strong>{live}</strong><small>场进行中</small><IconPulse size={18} /></div>
        <div className="overview-item"><span>模型就绪</span><strong>{analyzed}</strong><small>场已分析</small><IconSpark size={18} /></div>
        <Link className="overview-item link" to="/results"><span>今日完场</span><strong>{finished}</strong><small>进入复盘中心 →</small></Link>
      </div>
      <div className="toolbar">
        <div><b>赛事列表</b><span>{rows.length ? `共 ${rows.length} 场，点击赛事查看完整研判` : "等待今日赛程数据"}</span></div>
        <div className="switch" aria-label="赛事排序">
          <button className={sort === "num" ? "btn" : "btn ghost"} onClick={() => setSort("num")}>
            竞彩序号
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
            <><IconPulse size={24} /><b>等待今日赛程</b><span>数据服务正在同步赛事与市场信息</span></>
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
            <span>{m.home}</span><em>VS</em><span>{m.away}</span>
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
          <div><div className="block-h">模型赛果概率</div><ScanShape home={shape.homeWin} draw={shape.draw} away={shape.awayWin} over={shape.over25} /></div>
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

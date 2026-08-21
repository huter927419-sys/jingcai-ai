import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { fetchWeek, fmtKick, type WeekMatch } from "../api";
import Layout from "../Layout";
import { IconChart, IconClock, IconGrid, IconPulse, IconShield } from "../Icons";

const DAY_NAMES = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];

export default function Week() {
  const [matches, setMatches] = useState<WeekMatch[]>([]);
  const [range, setRange] = useState({ from: "", to: "" });
  const [filter, setFilter] = useState("all");
  const [err, setErr] = useState("");

  useEffect(() => {
    void fetchWeek().then((j) => {
      setMatches(j.matches);
      setRange({ from: j.from, to: j.to });
    }).catch((e) => setErr(e instanceof Error ? e.message : "加载失败"));
  }, []);

  const filtered = matches.filter((m) => filter === "all" || (filter === "finished" ? m.finished : m.businessDate === filter));
  const groups = useMemo(() => groupByDate(filtered), [filtered]);
  const days = [...new Set(matches.map((m) => m.businessDate))];
  const finished = matches.filter((m) => m.finished).length;
  const complete = matches.filter((m) => m.hasMarket && m.hasPreview && m.analysisCount > 0).length;

  return <Layout>
    <div className="page-head week-head">
      <div><div className="eyebrow"><span /> WEEKLY DATA ARCHIVE</div><h1>本周竞彩数据</h1><div className="sub">{range.from} 至 {range.to} · 本周赛程、盘口、阵容与研判统一归档</div></div>
    </div>
    <div className="week-summary">
      <div><IconGrid size={18}/><span>本周收录<strong>{matches.length}</strong><small>场竞彩赛事</small></span></div>
      <div><IconPulse size={18}/><span>数据完整<strong>{complete}</strong><small>盘口 · 阵容 · 研判</small></span></div>
      <div><IconShield size={18}/><span>已完场<strong>{finished}</strong><small>场完成对账</small></span></div>
      <div><IconChart size={18}/><span>待赛场次<strong>{matches.length - finished}</strong><small>持续更新临场数据</small></span></div>
    </div>
    <div className="week-filter">
      <button className={filter === "all" ? "active" : ""} onClick={() => setFilter("all")}>全部 {matches.length}</button>
      {days.map((d) => <button key={d} className={filter === d ? "active" : ""} onClick={() => setFilter(d)}>{dayLabel(d)} {matches.filter((m) => m.businessDate === d).length}</button>)}
      <button className={filter === "finished" ? "active" : ""} onClick={() => setFilter("finished")}>已完场 {finished}</button>
    </div>
    {err ? <div className="err">{err}</div> : null}
    {groups.map(([date, rows]) => <section className="week-group" key={date}>
      <div className="week-group-head"><div><b>{dayLabel(date)}</b><span>{date}</span></div><em>{rows.length} 场</em></div>
      <div className="week-table-head"><span>场次 / 联赛</span><span>对阵</span><span>开赛</span><span>数据完整度</span><span>状态</span></div>
      <div className="week-list">{rows.map((m) => <WeekRow key={m.id} match={m}/>)}</div>
    </section>)}
  </Layout>;
}

function WeekRow({ match: m }: { match: WeekMatch }) {
  const score = m.finished && m.homeGoals != null && m.awayGoals != null ? `${m.homeGoals}-${m.awayGoals}` : "VS";
  return <Link className="week-row" to={`/matches/${m.id}`}>
    <span className="week-id"><b>{m.numStr}</b><small>{m.leagueAbb || m.league}</small></span>
    <span className="week-teams"><b>{m.home}</b><em>{score}</em><b>{m.away}</b></span>
    <span className="week-kick"><IconClock size={12}/>{fmtKick(m.kickoff).slice(6)}</span>
    <span className="coverage"><i className={m.hasMarket ? "ok" : ""}>盘口</i><i className={m.hasPreview ? "ok" : ""}>阵容</i><i className={m.analysisCount > 0 ? "ok" : ""}>研判 {m.analysisCount || "—"}</i></span>
    <span className={`pill ${m.status === "临场" || m.status === "完场" ? "on" : ""}`}>{m.status}</span>
  </Link>;
}

function groupByDate(rows: WeekMatch[]): Array<[string, WeekMatch[]]> {
  const map = new Map<string, WeekMatch[]>();
  rows.forEach((m) => map.set(m.businessDate, [...(map.get(m.businessDate) ?? []), m]));
  return [...map];
}

function dayLabel(date: string): string {
  const d = new Date(`${date}T12:00:00`);
  return DAY_NAMES[d.getDay()] || date;
}

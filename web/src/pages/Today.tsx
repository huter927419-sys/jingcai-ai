import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { fetchToday, fmtKick, fmtPrice, playRows, sortMatches, type MatchRow, type MatchSort } from "../api";
import Layout from "../Layout";
import { ScanShape } from "../Charts";
import { IconBall, IconChart, IconClock, IconGauge, IconGoals, IconGrid, IconPulse, IconScale, IconShield, IconSpark } from "../Icons";

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
            基于赛程、市场赔率与多维共识生成的实时决策视图
          </div>
        </div>
        <div className="refresh-note"><i /><span>数据自动更新<small>每 20 秒同步一次</small></span></div>
      </div>
      <section className="value-flow" aria-label="产品分析闭环">
        <div className="value-flow-intro">
          <span>核心价值</span>
          <strong>从数据到决策，再到验证</strong>
        </div>
        <div className="value-step">
          <IconGrid size={18} />
          <span><b>多源数据</b><small>赛程 · 阵容 · 赔率</small></span>
          <em>01</em>
        </div>
        <div className="value-step">
          <IconSpark size={18} />
          <span><b>交叉研判</b><small>多维度独立判断</small></span>
          <em>02</em>
        </div>
        <div className="value-step signature">
          <IconScale size={18} />
          <span><b>价值研判™</b><small>本系统独有决策矩阵</small></span>
          <em>03</em>
        </div>
        <Link className="value-step" to="/results">
          <IconChart size={18} />
          <span><b>赛后验证</b><small>自动对账与排名</small></span>
          <em>04</em>
        </Link>
      </section>
      <section className="value-story">
        <div className="value-story-mark"><IconScale size={25} /><span>JC<small>VALUE MATRIX</small></span></div>
        <div className="value-story-copy">
          <span className="exclusive-tag">系统独有能力</span>
          <h2>看对方向，还要买在合适的价格</h2>
          <p>价值研判不是简单预测胜负。系统同步观察比赛概率、欧亚盘态度、资金拥挤与临场变化，识别“方向正确但价格不值”的常见陷阱。</p>
        </div>
        <div className="value-story-points">
          <div><b>01</b><span>先判断比赛方向<small>基本面与场上格局</small></span></div>
          <div><b>02</b><span>再检查市场定价<small>赔率、盘口与资金信号</small></span></div>
          <div><b>03</b><span>最后给执行等级<small>主推 · 可看 · 回避</small></span></div>
        </div>
        <div className="method-seal">专有方法论<small>权重与阈值不公开</small></div>
      </section>
      <div className="overview-grid">
        <div className="overview-item"><span>今日待赛</span><strong>{rows.length}</strong><small>场赛事</small><IconBall size={18} /></div>
        <div className="overview-item"><span>临场监测</span><strong>{live}</strong><small>场进行中</small><IconPulse size={18} /></div>
        <div className="overview-item"><span>研判就绪</span><strong>{analyzed}</strong><small>场已分析</small><IconSpark size={18} /></div>
        <Link className="overview-item link" to="/results"><span>今日完场</span><strong>{finished}</strong><small>进入复盘中心 →</small></Link>
      </div>
      <section className="reading-map" aria-label="分析阅读路径">
        <div className="reading-map-title"><span>READING PATH</span><h2>三步看懂一场研判</h2><p>首页先筛方向，详情页再核验证据与价格。</p></div>
        <div className="reading-step"><i><IconGauge size={18} /></i><div><em>01</em><b>先看方向强度</b><small>主方向与次选差距越大，比赛格局越清晰</small></div></div>
        <div className="reading-step"><i><IconGrid size={18} /></i><div><em>02</em><b>再核对支撑证据</b><small>结合票面变化、阵型首发与伤停信息</small></div></div>
        <div className="reading-step signature"><i><IconScale size={18} /></i><div><em>03</em><b>最后看价值等级</b><small>主推、可看、回避决定是否具备参考条件</small></div></div>
      </section>
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
              <Link to="/results">去复盘中心查看表现</Link>
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
  const digest = shape ? matchDigest(shape) : null;
  const ticketReady = [std?.h, std?.d, std?.a].filter((v) => Number(v) > 1).length;

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
          <div><div className="block-h">赛果概率</div><ScanShape home={shape.homeWin} draw={shape.draw} away={shape.awayWin} over={shape.over25} /></div>
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
      {digest ? (
        <div className="scan-insight">
          <div className="scan-insight-lead"><IconGauge size={16} /><span>主方向<strong>{digest.direction}</strong></span><b className={`strength ${digest.tone}`}>{digest.strength}</b></div>
          <div className="scan-indicator"><IconChart size={14} /><span>方向差</span><b>{digest.gap.toFixed(1)}%</b><i><em style={{ width: `${Math.min(100, digest.gap * 4)}%` }} /></i></div>
          <div className="scan-indicator"><IconGoals size={14} /><span>进球倾向</span><b>{shape!.over25 >= 52 ? "偏大" : shape!.over25 <= 48 ? "偏小" : "均衡"}</b><i><em style={{ width: `${shape!.over25}%` }} /></i></div>
          <div className="scan-indicator readiness"><IconShield size={14} /><span>票面完整</span><b>{ticketReady}/3</b><i><em style={{ width: `${(ticketReady / 3) * 100}%` }} /></i></div>
          <span className="scan-enter">查看完整研判 →</span>
        </div>
      ) : null}
    </Link>
  );
}

function matchDigest(shape: NonNullable<MatchRow["shape"]>): { direction: string; gap: number; strength: string; tone: string } {
  const sorted = [{ label: "主胜", p: shape.homeWin }, { label: "平局", p: shape.draw }, { label: "客胜", p: shape.awayWin }].sort((a, b) => b.p - a.p);
  const gap = sorted[0].p - sorted[1].p;
  if (gap >= 18) return { direction: sorted[0].label, gap, strength: "清晰", tone: "strong" };
  if (gap >= 8) return { direction: sorted[0].label, gap, strength: "倾向", tone: "medium" };
  return { direction: `${sorted[0].label} / ${sorted[1].label}`, gap, strength: "胶着", tone: "weak" };
}

import { useEffect, useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { IconBall, IconChart, IconGrid, IconLogo, IconPulse } from "./Icons";

type ExpertChip = { role: string };

export default function Layout({ children }: { children: React.ReactNode }) {
  const [experts, setExperts] = useState<ExpertChip[]>([]);

  useEffect(() => {
    void fetch("/api/health")
      .then((r) => r.json())
      .then((j) => {
        if (Array.isArray(j.experts) && j.experts.length) {
          setExperts(j.experts.map((x: { role: string }) => ({ role: x.role })));
          return;
        }
        setExperts((Array.isArray(j.models) ? j.models : []).map(() => ({ role: "专业研判" })));
      })
      .catch(() => setExperts([]));
  }, []);

  return (
    <>
      <header className="nav">
        <div className="nav-inner">
          <Link to="/" className="brand">
            <IconLogo />
            <span>
              JC Intelligence
              <small>竞彩决策分析终端</small>
            </span>
          </Link>
          <nav className="primary-nav" aria-label="主导航">
            <NavLink to="/" end className={({ isActive }) => (isActive ? "nav-link active" : "nav-link")}>
              今日赛事
            </NavLink>
            <NavLink to="/results" className={({ isActive }) => (isActive ? "nav-link active" : "nav-link")}>
              复盘中心
            </NavLink>
            <NavLink to="/week" className={({ isActive }) => (isActive ? "nav-link active" : "nav-link")}>
              本周数据
            </NavLink>
          </nav>
          <div className="system-state" title={experts.length ? experts.map((x) => x.role).join(" · ") : "本地分析引擎"}>
            <IconPulse size={15} />
            <span><b>研判引擎</b><small>{experts.length ? `${experts.length} 个维度就绪` : "本地模式"}</small></span>
            <i />
          </div>
        </div>
      </header>
      <div className="shell">
        {children}
        <footer className="analysis-disclaimer">
          <b>分析声明</b>
          <span>本站展示的盘口、概率、价值等级、情景比分及参考买入均基于有限数据和特定时间截面的分析，仅供研究参考，不构成任何结果或收益保证。数据延迟、临场变盘、阵容调整及比赛偶发事件均可能导致判断失效，请独立决策，相关风险与结果由使用者自行承担。</span>
        </footer>
      </div>
      <nav className="mobile-nav" aria-label="移动端主导航">
        <NavLink to="/" end className={({ isActive }) => (isActive ? "active" : "")}>
          <IconBall size={19} />
          <span>今日</span>
        </NavLink>
        <NavLink to="/results" className={({ isActive }) => (isActive ? "active" : "")}>
          <IconChart size={19} />
          <span>复盘</span>
        </NavLink>
        <NavLink to="/week" className={({ isActive }) => (isActive ? "active" : "")}>
          <IconGrid size={19} />
          <span>本周</span>
        </NavLink>
      </nav>
    </>
  );
}

import { useEffect, useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { IconLogo, IconPulse } from "./Icons";

type ExpertChip = { name: string; role: string };

export default function Layout({ children }: { children: React.ReactNode }) {
  const [experts, setExperts] = useState<ExpertChip[]>([]);

  useEffect(() => {
    void fetch("/api/health")
      .then((r) => r.json())
      .then((j) => {
        if (Array.isArray(j.experts) && j.experts.length) {
          setExperts(j.experts.map((x: { name: string; role: string }) => ({ name: x.name, role: x.role })));
          return;
        }
        setExperts((Array.isArray(j.models) ? j.models : []).map((n: string) => ({ name: n, role: n })));
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
          </nav>
          <div className="system-state" title={experts.length ? experts.map((x) => x.role).join(" · ") : "本地分析引擎"}>
            <IconPulse size={15} />
            <span><b>分析引擎</b><small>{experts.length ? `${experts.length} 位专家在线` : "本地模式"}</small></span>
            <i />
          </div>
        </div>
      </header>
      <div className="shell">{children}</div>
    </>
  );
}

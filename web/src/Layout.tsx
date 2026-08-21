import { useEffect, useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { IconLogo } from "./Icons";

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
              竞彩分析
              <small>今日看板 · 完场看结果</small>
            </span>
          </Link>
          <div className="model-live">
            <NavLink to="/" end className={({ isActive }) => (isActive ? "hud-chip" : "hud-chip dim")}>
              今日
            </NavLink>
            <NavLink to="/results" className={({ isActive }) => (isActive ? "hud-chip" : "hud-chip dim")}>
              结果
            </NavLink>
            {experts.length ? (
              experts.map((n) => (
                <span className="hud-chip dim" key={n.name} title={n.name}>
                  {n.role}
                </span>
              ))
            ) : (
              <span className="hud-chip dim">LOCAL</span>
            )}
          </div>
        </div>
      </header>
      <div className="shell">{children}</div>
    </>
  );
}

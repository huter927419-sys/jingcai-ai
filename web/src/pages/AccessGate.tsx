import { FormEvent, ReactNode, useEffect, useState } from "react";

type Status = { authorized: boolean; reason?: string; durationDays?: number; expiresAt?: string };

export default function AccessGate({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<Status | null>(null);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const load = () => fetch("/api/access/status").then(r => r.json()).then(setStatus).catch(() => setStatus({ authorized: false, reason: "missing" }));
  useEffect(() => { load(); const timer = window.setInterval(load, 10_000); return () => window.clearInterval(timer); }, []);
  if (!status) return <div className="access-loading">正在准备今日数据…</div>;
  if (status.authorized) return <>{children}</>;
  async function redeem(e: FormEvent) {
    e.preventDefault(); setBusy(true); setError("");
    try { const r = await fetch("/api/access/redeem", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ code }) }); const j = await r.json(); if (!r.ok) throw new Error(j.error || "访问码无效"); await load(); setCode(""); } catch (e) { setError(e instanceof Error ? e.message : "访问码无效或已使用"); } finally { setBusy(false); }
  }
  return <div className="access-gate"><div className="access-background" aria-hidden="true">{children}</div><div className="access-shade" /><section className="access-card" role="dialog" aria-modal="true"><div className="access-mark">JC</div><p className="eyebrow">JINGCAI INSIGHT</p><h1>输入访问码，继续查看</h1><p className="access-copy">本系统按授权周期开放专业赛事数据。输入有效访问码后即可解锁完整赛程、盘口与价值研判。</p><form onSubmit={redeem}><label htmlFor="access-code">访问码</label><input id="access-code" value={code} onChange={e => setCode(e.target.value.toUpperCase())} placeholder="例如 7KX4P-N9Q2M" autoComplete="one-time-code" maxLength={11} /><button className="btn primary" disabled={busy || code.replace(/[- ]/g, "").length !== 10}>{busy ? "验证中…" : "验证并进入"}</button>{error && <div className="access-error">{error}</div>}</form><small>访问权限到期后，需要输入新的访问码。请合理安排使用周期。</small></section></div>;
}

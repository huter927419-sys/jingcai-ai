import { FormEvent, useEffect, useMemo, useState } from "react";

type Code = { id: number; code: string; durationDays: number; status: string; createdAt: string; activatedAt?: string; expiresAt?: string; terminatedAt?: string; lastSeenAt?: string; useCount: number };
type Pool = { durationDays: number; total: number; unused: number; active: number; expired: number; terminated: number };
const labels: Record<string, string> = { unused: "未使用", active: "使用中", expired: "已到期", terminated: "已终止" };
function time(v?: string) { return v ? new Date(v).toLocaleString("zh-CN", { hour12: false }) : "—"; }

function pageItems(current: number, pages: number): (number | "…")[] {
  if (pages <= 7) return Array.from({ length: pages }, (_, i) => i + 1);
  const set = new Set([1, pages, current, current - 1, current + 1]);
  const nums = [...set].filter(n => n >= 1 && n <= pages).sort((a, b) => a - b);
  const out: (number | "…")[] = [];
  let prev = 0;
  for (const n of nums) {
    if (prev && n - prev > 1) out.push("…");
    out.push(n);
    prev = n;
  }
  return out;
}

export default function AdminAccess() {
  const [auth, setAuth] = useState<boolean | null>(null);
  const [codes, setCodes] = useState<Code[]>([]);
  const [pools, setPools] = useState<Pool[]>([]);
  const [total, setTotal] = useState(0);
  const [pages, setPages] = useState(1);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [filter, setFilter] = useState({ days: "", status: "unused", q: "" });
  const [qInput, setQInput] = useState("");
  const [login, setLogin] = useState({ username: "", password: "" });
  const [message, setMessage] = useState("");

  useEffect(() => {
    const t = window.setTimeout(() => {
      setFilter(f => {
        if (f.q === qInput) return f;
        setPage(1);
        return { ...f, q: qInput };
      });
    }, 300);
    return () => window.clearTimeout(t);
  }, [qInput]);

  const query = useMemo(() => {
    const p = new URLSearchParams();
    if (filter.days) p.set("days", filter.days);
    if (filter.status) p.set("status", filter.status);
    if (filter.q) p.set("q", filter.q);
    p.set("page", String(page));
    p.set("pageSize", String(pageSize));
    return p.toString();
  }, [filter, page, pageSize]);

  const load = () => fetch("/api/admin/status").then(async r => {
    const text = await r.text();
    if (!r.ok) throw new Error(`状态接口 ${r.status}`);
    let j: { authenticated?: boolean };
    try { j = JSON.parse(text); } catch { throw new Error("后端尚未更新，请重启服务"); }
    setAuth(Boolean(j.authenticated));
    if (!j.authenticated) return;
    const data = await fetch(`/api/admin/access-codes?${query}`).then(r => r.json());
    setCodes(data.codes || []);
    setPools(data.pools || []);
    setTotal(Number(data.total) || 0);
    setPages(Math.max(1, Number(data.pages) || 1));
    if (data.page && data.page !== page) setPage(data.page);
  }).catch(e => { setAuth(false); setMessage(e instanceof Error ? e.message : "无法连接后端"); });

  useEffect(() => { load(); }, [auth, query]);

  function changeFilter(next: Partial<typeof filter>) {
    setFilter(f => ({ ...f, ...next }));
    setPage(1);
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    setMessage("");
    const r = await fetch("/api/admin/login", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(login) });
    if (!r.ok) { setMessage("账号或密码错误"); return; }
    setAuth(true);
    setLogin({ username: "", password: "" });
  }
  async function terminate(id: number) {
    if (!confirm("确定终止此访问码？已有会话将立即失效。")) return;
    await fetch(`/api/admin/access-codes/${id}/terminate`, { method: "POST" });
    load();
  }
  async function generate(days: number) {
    await fetch("/api/admin/access-codes/generate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ durationDays: days, count: 1000 }) });
    setMessage(`已确保 ${days} 天访问码池不少于 1000 份`);
    load();
  }

  if (auth === null) return <div className="admin-shell">正在验证管理员会话…</div>;
  if (!auth) return (
    <main className="admin-shell">
      <form className="admin-login" onSubmit={submit}>
        <p className="eyebrow">PRIVATE CONSOLE</p>
        <h1>管理员登录</h1>
        <p>请输入管理员账号和密码。</p>
        <input placeholder="管理员账号" value={login.username} onChange={e => setLogin({ ...login, username: e.target.value })} />
        <input type="password" placeholder="管理员密码" value={login.password} onChange={e => setLogin({ ...login, password: e.target.value })} />
        <button className="btn primary">登录</button>
        {message && <div className="access-error">{message}</div>}
      </form>
    </main>
  );

  const from = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const to = Math.min(total, page * pageSize);

  return (
    <main className="admin-shell">
      <header className="admin-head">
        <div>
          <p className="eyebrow">PRIVATE CONSOLE</p>
          <h1>访问码管理</h1>
        </div>
        <button className="btn" onClick={() => { fetch("/api/admin/logout", { method: "POST" }); setAuth(false); }}>退出</button>
      </header>
      <div className="admin-pools">
        {(pools.length ? pools : [3, 7, 15, 30].map(d => ({ durationDays: d, total: 0, unused: 0, active: 0, expired: 0, terminated: 0 }))).map(p => (
          <button key={p.durationDays} className={`admin-pool${filter.days === String(p.durationDays) ? " on" : ""}`} onClick={() => changeFilter({ days: filter.days === String(p.durationDays) ? "" : String(p.durationDays) })}>
            <span>{p.durationDays} 天授权</span>
            <b>{p.unused}</b>
            <small>未使用 / 共 {p.total}{p.active ? ` · 在用 ${p.active}` : ""}</small>
          </button>
        ))}
      </div>
      <div className="admin-toolbar">
        <select value={filter.days} onChange={e => changeFilter({ days: e.target.value })}>
          <option value="">全部周期</option>
          <option value="3">3 天</option>
          <option value="7">7 天</option>
          <option value="15">15 天</option>
          <option value="30">30 天</option>
        </select>
        <select value={filter.status} onChange={e => changeFilter({ status: e.target.value })}>
          <option value="">全部状态</option>
          {Object.entries(labels).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
        </select>
        <input placeholder="搜索访问码" value={qInput} onChange={e => setQInput(e.target.value.toUpperCase())} />
        <select value={pageSize} onChange={e => { setPageSize(Number(e.target.value)); setPage(1); }}>
          <option value={20}>每页 20</option>
          <option value={50}>每页 50</option>
          <option value={100}>每页 100</option>
        </select>
        {[3, 7, 15, 30].map(d => <button className="btn" key={d} onClick={() => generate(d)}>补充 {d} 天</button>)}
      </div>
      {message && <p className="admin-message">{message}</p>}
      <p className="admin-count">共 {total} 条，当前 {from}-{to}，第 {page} / {pages} 页</p>
      <div className="admin-table-wrap">
        <table>
          <thead>
            <tr><th>访问码</th><th>周期</th><th>状态</th><th>首次使用</th><th>到期时间</th><th>最近访问</th><th>操作</th></tr>
          </thead>
          <tbody>
            {codes.length === 0 ? (
              <tr><td colSpan={7} className="admin-empty">没有符合条件的访问码</td></tr>
            ) : codes.map(c => (
              <tr key={c.id}>
                <td><code>{c.code}</code></td>
                <td>{c.durationDays} 天</td>
                <td><span className={`admin-status ${c.status}`}>{labels[c.status] || c.status}</span></td>
                <td>{time(c.activatedAt)}</td>
                <td>{time(c.expiresAt)}</td>
                <td>{time(c.lastSeenAt)}</td>
                <td>{c.status === "active" && <button className="link-danger" onClick={() => terminate(c.id)}>终止</button>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="admin-pager">
        <button className="btn" disabled={page <= 1} onClick={() => setPage(1)}>首页</button>
        <button className="btn" disabled={page <= 1} onClick={() => setPage(p => Math.max(1, p - 1))}>上一页</button>
        {pageItems(page, pages).map((n, i) => n === "…" ? <span key={`e${i}`}>…</span> : (
          <button key={n} className={`btn${n === page ? " primary" : ""}`} onClick={() => setPage(n)}>{n}</button>
        ))}
        <button className="btn" disabled={page >= pages} onClick={() => setPage(p => Math.min(pages, p + 1))}>下一页</button>
        <button className="btn" disabled={page >= pages} onClick={() => setPage(pages)}>末页</button>
      </div>
    </main>
  );
}

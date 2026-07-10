import { createFileRoute } from "@tanstack/react-router";
import { motion, AnimatePresence } from "framer-motion";
import { useEffect, useState } from "react";
import {
  Activity, Search, Bell, ChevronRight, Cpu, HardDrive, Network,
  LayoutGrid, Server, ScrollText, Settings, GitBranch, Gauge, PanelLeft, Command,
} from "lucide-react";
import { ClusterViz } from "@/components/cluster-viz";
import { Link } from "@tanstack/react-router";

export const Route = createFileRoute("/dashboard")({
  head: () => ({
    meta: [
      { title: "Console — Raftline" },
      { name: "description", content: "Live cluster monitoring, replication timeline, and consensus activity." },
    ],
  }),
  component: Dashboard,
});

function Dashboard() {
  const [collapsed, setCollapsed] = useState(false);
  return (
    <div className="flex min-h-screen bg-surface text-foreground">
      <Sidebar collapsed={collapsed} onToggle={() => setCollapsed((c) => !c)} />
      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar />
        <main className="flex-1 space-y-6 p-6 pt-8">
          <ClusterOverview />
          <div className="grid gap-6 lg:grid-cols-3">
            <div className="lg:col-span-2 space-y-6">
              <TopologyCard />
              <ReplicationTimeline />
            </div>
            <div className="space-y-6">
              <MetricsPanel />
              <ConsensusFeed />
            </div>
          </div>
          <div className="grid gap-6 lg:grid-cols-3">
            <NodeCards />
          </div>
          <LogExplorer />
        </main>
      </div>
    </div>
  );
}

/* -------- Sidebar -------- */
function Sidebar({ collapsed, onToggle }: { collapsed: boolean; onToggle: () => void }) {
  const items = [
    { icon: LayoutGrid, label: "Overview", active: true },
    { icon: Server, label: "Nodes" },
    { icon: GitBranch, label: "Replication" },
    { icon: Gauge, label: "Metrics" },
    { icon: ScrollText, label: "Logs" },
    { icon: Activity, label: "Events" },
    { icon: Settings, label: "Settings" },
  ];
  return (
    <motion.aside
      animate={{ width: collapsed ? 72 : 244 }}
      transition={{ type: "spring", stiffness: 180, damping: 22 }}
      className="sticky top-0 flex h-screen shrink-0 flex-col border-r border-border bg-sidebar"
    >
      <div className="flex h-14 items-center gap-2.5 px-4">
        <Link to="/" className="flex h-7 w-7 items-center justify-center rounded-lg bg-foreground">
          <div className="h-3 w-3 rounded-sm bg-background" />
        </Link>
        <AnimatePresence>
          {!collapsed && (
            <motion.span
              initial={{ opacity: 0, x: -6 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -6 }}
              className="text-display text-[13.5px] font-semibold tracking-tight"
            >
              Raftline
            </motion.span>
          )}
        </AnimatePresence>
      </div>
      <div className="mt-2 flex-1 space-y-0.5 px-2">
        {items.map((it) => (
          <button
            key={it.label}
            className={`group flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-[13px] font-medium transition-colors ${
              it.active ? "bg-accent text-foreground" : "text-muted-foreground hover:bg-accent/60 hover:text-foreground"
            }`}
          >
            <it.icon className="h-4 w-4 shrink-0" strokeWidth={1.75} />
            {!collapsed && <span className="truncate">{it.label}</span>}
          </button>
        ))}
      </div>
      <button
        onClick={onToggle}
        className="mx-3 mb-3 flex items-center justify-center rounded-lg border border-border bg-surface-elevated/60 py-2 text-muted-foreground hover:text-foreground"
      >
        <PanelLeft className="h-3.5 w-3.5" />
      </button>
    </motion.aside>
  );
}

/* -------- Top bar -------- */
function TopBar() {
  return (
    <div className="sticky top-0 z-20 flex h-14 items-center gap-3 border-b border-border bg-background/70 px-6 backdrop-blur-xl">
      <div className="flex items-center gap-2 text-[13px] text-muted-foreground">
        <span>Cluster</span>
        <ChevronRight className="h-3.5 w-3.5" />
        <span className="text-foreground">production-us-east</span>
      </div>
      <div className="mx-4 flex flex-1 items-center gap-2 rounded-lg border border-border bg-surface-elevated/60 px-3 py-1.5">
        <Search className="h-3.5 w-3.5 text-muted-foreground" />
        <input className="flex-1 bg-transparent text-[13px] outline-none placeholder:text-muted-foreground" placeholder="Search keys, nodes, events…" />
        <kbd className="flex items-center gap-0.5 rounded border border-border px-1.5 py-0.5 text-[10px] text-muted-foreground">
          <Command className="h-2.5 w-2.5" />K
        </kbd>
      </div>
      <button className="rounded-lg p-2 text-muted-foreground hover:bg-accent hover:text-foreground">
        <Bell className="h-4 w-4" />
      </button>
      <div className="h-7 w-7 rounded-full bg-gradient-to-br from-primary to-chart-4" />
    </div>
  );
}

/* -------- Overview -------- */
function useCounter(target: number, duration = 1200) {
  const [v, setV] = useState(0);
  useEffect(() => {
    let raf = 0; const start = performance.now();
    const tick = (t: number) => {
      const p = Math.min(1, (t - start) / duration);
      const eased = 1 - Math.pow(1 - p, 3);
      setV(target * eased);
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, duration]);
  return v;
}

function ClusterOverview() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
      transition={{ type: "spring", stiffness: 120, damping: 20 }}
      className="card-elevated relative overflow-hidden p-6"
    >
      <div className="pointer-events-none absolute inset-0 mesh-bg opacity-40" />
      <div className="relative flex flex-wrap items-end justify-between gap-6">
        <div>
          <div className="flex items-center gap-2">
            <span className="h-1.5 w-1.5 rounded-full bg-success breathe" />
            <span className="text-[12px] font-medium uppercase tracking-wider text-muted-foreground">Cluster healthy</span>
          </div>
          <h1 className="text-display mt-2 text-3xl font-semibold tracking-tight">production-us-east</h1>
          <div className="mt-1.5 text-[13px] text-muted-foreground">5 nodes · Term 42 · Leader Node 01 · Uptime 27d 14h</div>
        </div>
        <div className="flex gap-3">
          <Stat label="Throughput" value={useCounter(18420)} suffix=" ops/s" />
          <Stat label="Commit p50" value={useCounter(4.2, 900)} suffix=" ms" decimals={1} />
          <Stat label="Availability" value={useCounter(99.999, 1400)} suffix="%" decimals={3} />
        </div>
      </div>
    </motion.div>
  );
}

function Stat({ label, value, suffix, decimals = 0 }: { label: string; value: number; suffix?: string; decimals?: number }) {
  return (
    <div className="min-w-[140px] rounded-2xl border border-border bg-surface-elevated/70 px-4 py-3 backdrop-blur">
      <div className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className="text-display mt-1 text-2xl font-semibold tabular-nums tracking-tight">
        {value.toLocaleString(undefined, { minimumFractionDigits: decimals, maximumFractionDigits: decimals })}
        <span className="text-[13px] font-medium text-muted-foreground">{suffix}</span>
      </div>
    </div>
  );
}

/* -------- Topology -------- */
function TopologyCard() {
  return (
    <div className="card-elevated overflow-hidden">
      <CardHeader title="Cluster Topology" subtitle="Live consensus and replication activity" />
      <div className="relative flex items-center justify-center bg-surface p-6">
        <ClusterViz size={440} />
      </div>
    </div>
  );
}

/* -------- Replication timeline -------- */
function ReplicationTimeline() {
  const entries = Array.from({ length: 14 }, (_, i) => ({
    idx: 1042 - i,
    status: i === 0 ? "pending" : i < 3 ? "committed" : "applied",
    latency: 2 + Math.random() * 6,
    op: ["SET user.42.session", "DEL cache.9821", "SET order.19923.state", "SET metrics.tick"][i % 4],
  }));
  return (
    <div className="card-elevated">
      <CardHeader title="Replication Timeline" subtitle="Last 14 log entries" />
      <div className="divide-y divide-border">
        {entries.map((e, i) => (
          <motion.div
            key={e.idx}
            initial={{ opacity: 0, x: -8 }} animate={{ opacity: 1, x: 0 }}
            transition={{ delay: i * 0.02 }}
            className="flex items-center gap-4 px-6 py-2.5 hover:bg-accent/40"
          >
            <div className="w-16 font-mono text-[12px] tabular-nums text-muted-foreground">#{e.idx}</div>
            <StatusDot status={e.status} />
            <div className="w-24 text-[12px] capitalize text-muted-foreground">{e.status}</div>
            <div className="flex-1 truncate font-mono text-[12.5px] text-foreground/85">{e.op}</div>
            <div className="w-16 text-right font-mono text-[12px] tabular-nums text-muted-foreground">{e.latency.toFixed(1)}ms</div>
            <ReplicationBar status={e.status} />
          </motion.div>
        ))}
      </div>
    </div>
  );
}

function ReplicationBar({ status }: { status: string }) {
  const pct = status === "pending" ? 60 : status === "committed" ? 100 : 100;
  return (
    <div className="w-32">
      <div className="h-1 overflow-hidden rounded-full bg-accent">
        <motion.div
          initial={{ width: 0 }} animate={{ width: `${pct}%` }}
          transition={{ type: "spring", stiffness: 100, damping: 24 }}
          className="h-full rounded-full bg-primary"
        />
      </div>
    </div>
  );
}

function StatusDot({ status }: { status: string }) {
  const color = status === "pending" ? "bg-warning" : status === "committed" ? "bg-primary" : "bg-success";
  return <span className={`h-1.5 w-1.5 rounded-full ${color}`} />;
}

/* -------- Metrics panel -------- */
function MetricsPanel() {
  return (
    <div className="card-elevated">
      <CardHeader title="Performance" subtitle="Rolling 5 minutes" />
      <div className="space-y-5 px-6 pb-6">
        <Sparkline label="Throughput" unit="ops/s" color="var(--primary)" />
        <Sparkline label="Commit latency" unit="ms" color="var(--chart-2)" seed={2} />
        <Sparkline label="Network I/O" unit="MB/s" color="var(--chart-4)" seed={3} />
      </div>
    </div>
  );
}

function Sparkline({ label, unit, color, seed = 1 }: { label: string; unit: string; color: string; seed?: number }) {
  const points = Array.from({ length: 40 }, (_, i) => {
    return 0.5 + 0.3 * Math.sin(i * 0.4 + seed) + 0.15 * Math.sin(i * 1.1 + seed * 2);
  });
  const w = 260, h = 56;
  const path = points.map((p, i) => `${i === 0 ? "M" : "L"} ${(i / (points.length - 1)) * w} ${h - p * h}`).join(" ");
  const area = `${path} L ${w} ${h} L 0 ${h} Z`;
  const current = (points[points.length - 1] * 100).toFixed(1);
  return (
    <div>
      <div className="flex items-baseline justify-between">
        <div className="text-[12px] font-medium text-muted-foreground">{label}</div>
        <div className="text-display text-[15px] font-semibold tabular-nums">
          {current}<span className="ml-1 text-[11px] font-medium text-muted-foreground">{unit}</span>
        </div>
      </div>
      <svg viewBox={`0 0 ${w} ${h}`} className="mt-1.5 h-14 w-full">
        <defs>
          <linearGradient id={`g-${label}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.28" />
            <stop offset="100%" stopColor={color} stopOpacity="0" />
          </linearGradient>
        </defs>
        <motion.path
          d={area} fill={`url(#g-${label})`}
          initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.8 }}
        />
        <motion.path
          d={path} fill="none" stroke={color} strokeWidth={1.5} strokeLinecap="round"
          initial={{ pathLength: 0 }} animate={{ pathLength: 1 }}
          transition={{ duration: 1.2, ease: [0.22, 1, 0.36, 1] }}
        />
      </svg>
    </div>
  );
}

/* -------- Consensus feed -------- */
function ConsensusFeed() {
  const events = [
    { t: "12:04:22", label: "AppendEntries acked", detail: "Node 03 · idx 1042", kind: "info" },
    { t: "12:04:21", label: "Commit index advanced", detail: "→ 1041", kind: "ok" },
    { t: "12:04:19", label: "Heartbeat", detail: "Term 42 · 5/5", kind: "info" },
    { t: "12:03:58", label: "Snapshot installed", detail: "Node 05 · 128 MB", kind: "ok" },
    { t: "12:03:41", label: "Vote granted", detail: "Node 02 → Node 01", kind: "info" },
  ];
  return (
    <div className="card-elevated">
      <CardHeader title="Consensus Activity" subtitle="Real-time protocol events" />
      <div className="space-y-1 px-3 pb-4">
        {events.map((e, i) => (
          <motion.div
            key={i} initial={{ opacity: 0, y: 4 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}
            className="flex items-start gap-3 rounded-lg px-3 py-2 hover:bg-accent/50"
          >
            <div className={`mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full ${e.kind === "ok" ? "bg-success" : "bg-primary"}`} />
            <div className="min-w-0 flex-1">
              <div className="text-[13px] font-medium text-foreground">{e.label}</div>
              <div className="text-[11.5px] text-muted-foreground">{e.detail}</div>
            </div>
            <div className="font-mono text-[11px] tabular-nums text-muted-foreground">{e.t}</div>
          </motion.div>
        ))}
      </div>
    </div>
  );
}

/* -------- Node cards -------- */
function NodeCards() {
  const nodes = [
    { name: "Node 01", role: "Leader", cpu: 34, mem: 58, disk: 42, net: 128 },
    { name: "Node 02", role: "Follower", cpu: 22, mem: 47, disk: 41, net: 96 },
    { name: "Node 03", role: "Follower", cpu: 27, mem: 51, disk: 43, net: 104 },
  ];
  return (
    <>
      {nodes.map((n, i) => (
        <motion.div
          key={n.name}
          initial={{ opacity: 0, y: 12 }} whileInView={{ opacity: 1, y: 0 }} viewport={{ once: true }}
          transition={{ delay: i * 0.06, type: "spring", stiffness: 140, damping: 20 }}
          whileHover={{ y: -2 }}
          className="card-elevated p-5"
        >
          <div className="flex items-start justify-between">
            <div>
              <div className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full bg-success breathe" />
                <span className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">{n.role}</span>
              </div>
              <div className="text-display mt-1 text-[16px] font-semibold tracking-tight">{n.name}</div>
              <div className="text-[11.5px] text-muted-foreground">us-east-1a · 10.0.4.{12 + i}</div>
            </div>
            <div className="rounded-full border border-border bg-surface px-2 py-0.5 font-mono text-[10.5px] text-muted-foreground">
              v1.4.2
            </div>
          </div>
          <div className="mt-4 grid grid-cols-2 gap-3">
            <Meter icon={Cpu} label="CPU" value={n.cpu} suffix="%" />
            <Meter icon={HardDrive} label="Memory" value={n.mem} suffix="%" />
            <Meter icon={HardDrive} label="Disk" value={n.disk} suffix="%" />
            <Meter icon={Network} label="Network" value={n.net} suffix=" MB/s" max={200} />
          </div>
        </motion.div>
      ))}
    </>
  );
}

function Meter({ icon: Icon, label, value, suffix, max = 100 }: { icon: typeof Cpu; label: string; value: number; suffix: string; max?: number }) {
  const pct = (value / max) * 100;
  return (
    <div className="rounded-xl bg-surface p-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
          <Icon className="h-3 w-3" strokeWidth={1.75} />
          {label}
        </div>
        <div className="font-mono text-[11.5px] tabular-nums text-foreground/80">{value}{suffix}</div>
      </div>
      <div className="mt-2 h-1 overflow-hidden rounded-full bg-accent">
        <motion.div
          initial={{ width: 0 }} whileInView={{ width: `${pct}%` }} viewport={{ once: true }}
          transition={{ duration: 1, ease: [0.22, 1, 0.36, 1] }}
          className="h-full rounded-full bg-primary"
        />
      </div>
    </div>
  );
}

/* -------- Log explorer -------- */
function LogExplorer() {
  const logs = [
    { t: "12:04:22.481", lvl: "INFO", src: "raft", msg: "AppendEntries success node=03 term=42 idx=1042" },
    { t: "12:04:22.412", lvl: "INFO", src: "raft", msg: "leader heartbeat quorum=5/5" },
    { t: "12:04:21.998", lvl: "INFO", src: "store", msg: "commit index advanced 1040 → 1041" },
    { t: "12:04:20.221", lvl: "WARN", src: "net", msg: "peer node=05 rtt=118ms above threshold=90ms" },
    { t: "12:04:18.004", lvl: "INFO", src: "raft", msg: "snapshot installed node=05 bytes=134217728" },
    { t: "12:04:14.771", lvl: "DEBUG", src: "wal", msg: "fsync 2 pages 512B" },
  ];
  const color = (l: string) => l === "WARN" ? "text-warning" : l === "ERROR" ? "text-destructive" : l === "DEBUG" ? "text-muted-foreground" : "text-primary";
  return (
    <div className="card-elevated">
      <div className="flex items-center justify-between border-b border-border p-4">
        <div>
          <div className="text-display text-[15px] font-semibold tracking-tight">Log Explorer</div>
          <div className="text-[12px] text-muted-foreground">Streaming from all 5 nodes</div>
        </div>
        <div className="flex items-center gap-2">
          <span className="flex items-center gap-1.5 rounded-full border border-border px-2.5 py-1 text-[11px] text-muted-foreground">
            <span className="h-1.5 w-1.5 rounded-full bg-success breathe" /> Live
          </span>
        </div>
      </div>
      <div className="max-h-80 overflow-y-auto p-2 font-mono text-[12px] leading-relaxed">
        {logs.map((l, i) => (
          <motion.div
            key={i} initial={{ opacity: 0, x: -6 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: i * 0.03 }}
            className="flex gap-4 rounded-md px-3 py-1 hover:bg-accent/40"
          >
            <span className="w-24 shrink-0 text-muted-foreground">{l.t}</span>
            <span className={`w-14 shrink-0 font-semibold ${color(l.lvl)}`}>{l.lvl}</span>
            <span className="w-12 shrink-0 text-muted-foreground">{l.src}</span>
            <span className="min-w-0 text-foreground/85">{l.msg}</span>
          </motion.div>
        ))}
      </div>
    </div>
  );
}

function CardHeader({ title, subtitle }: { title: string; subtitle?: string }) {
  return (
    <div className="border-b border-border p-5">
      <div className="text-display text-[15px] font-semibold tracking-tight">{title}</div>
      {subtitle && <div className="mt-0.5 text-[12px] text-muted-foreground">{subtitle}</div>}
    </div>
  );
}

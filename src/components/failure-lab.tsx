import { motion, AnimatePresence } from "framer-motion";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ClusterViz } from "@/components/cluster-viz";
import { Zap, Scissors, RotateCcw, PlayCircle } from "lucide-react";

type Role = "leader" | "follower" | "candidate" | "down";
interface SimNode {
  id: string;
  label: string;
  role: Role;
  x: number;
  y: number;
  side: "a" | "b";
}

interface Event {
  id: number;
  t: number;
  kind: "info" | "warn" | "ok" | "err";
  label: string;
  detail?: string;
}

const SIZE = 440;

function buildInitial(): SimNode[] {
  const cx = SIZE / 2;
  const cy = SIZE / 2;
  const r = SIZE * 0.32;
  const followers = 4;
  const nodes: SimNode[] = [
    { id: "n0", label: "Node 01", role: "leader", x: cx, y: cy, side: "a" },
  ];
  for (let i = 0; i < followers; i++) {
    const a = (i / followers) * Math.PI * 2 - Math.PI / 2;
    const x = cx + Math.cos(a) * r;
    nodes.push({
      id: `n${i + 1}`,
      label: `Node 0${i + 2}`,
      role: "follower",
      x,
      y: cy + Math.sin(a) * r,
      side: x < cx ? "a" : "b",
    });
  }
  return nodes;
}

export function FailureLab() {
  const [nodes, setNodes] = useState<SimNode[]>(() => buildInitial());
  const [partition, setPartition] = useState(false);
  const [term, setTerm] = useState(42);
  const [status, setStatus] = useState<"healthy" | "electing" | "partitioned" | "recovering">("healthy");
  const [busy, setBusy] = useState(false);
  const [events, setEvents] = useState<Event[]>([
    { id: 0, t: Date.now(), kind: "ok", label: "Cluster steady", detail: "Term 42 · 5/5 quorum" },
  ]);
  const idRef = useRef(1);
  const timers = useRef<number[]>([]);

  useEffect(() => () => timers.current.forEach((t) => clearTimeout(t)), []);
  const wait = (ms: number) =>
    new Promise<void>((resolve) => {
      const id = window.setTimeout(() => resolve(), ms);
      timers.current.push(id);
    });

  const emit = useCallback((e: Omit<Event, "id" | "t">) => {
    setEvents((prev) => [{ id: idRef.current++, t: Date.now(), ...e }, ...prev].slice(0, 8));
  }, []);

  const reset = useCallback(async () => {
    if (busy) return;
    setBusy(true);
    setPartition(false);
    setNodes(buildInitial());
    setStatus("healthy");
    emit({ kind: "ok", label: "Cluster restored", detail: "All 5 nodes online" });
    await wait(400);
    setBusy(false);
  }, [busy, emit]);

  const crashLeader = useCallback(async () => {
    if (busy) return;
    setBusy(true);
    setStatus("electing");

    const leader = nodes.find((n) => n.role === "leader");
    if (!leader) {
      setBusy(false);
      return;
    }
    emit({ kind: "err", label: `${leader.label} lost`, detail: "Heartbeat timeout · leader down" });
    setNodes((prev) => prev.map((n) => (n.id === leader.id ? { ...n, role: "down" } : n)));

    await wait(650);

    // Election: pick two candidates
    const alive = nodes.filter((n) => n.id !== leader.id);
    const candidates = alive.slice(0, 2).map((n) => n.id);
    emit({ kind: "warn", label: "Election started", detail: `Term ${term + 1} · ${candidates.length} candidates` });
    setNodes((prev) =>
      prev.map((n) =>
        candidates.includes(n.id) ? { ...n, role: "candidate" } : n,
      ),
    );

    await wait(900);

    // Vote gathering
    emit({ kind: "info", label: "RequestVote broadcast", detail: `${candidates[0]} → peers` });
    await wait(700);
    emit({ kind: "ok", label: "Quorum reached", detail: `3 votes for ${alive[0].label}` });

    const winner = alive[0].id;
    setTerm((t) => t + 1);
    setNodes((prev) =>
      prev.map((n) => {
        if (n.id === winner) return { ...n, role: "leader" };
        if (candidates.includes(n.id)) return { ...n, role: "follower" };
        return n;
      }),
    );
    emit({ kind: "ok", label: `${alive[0].label} promoted`, detail: `New leader · term ${term + 1}` });

    await wait(500);
    setStatus("healthy");
    setBusy(false);
  }, [busy, nodes, emit, term]);

  const partitionNetwork = useCallback(async () => {
    if (busy) return;
    setBusy(true);
    setStatus("partitioned");
    setPartition(true);
    emit({ kind: "warn", label: "Network partition", detail: "Link severed between A ↔ B" });

    await wait(1200);
    const leader = nodes.find((n) => n.role === "leader");
    if (leader) {
      const minoritySide = leader.side === "a" ? "b" : "a";
      const minority = nodes.filter((n) => n.side === minoritySide);
      emit({
        kind: "warn",
        label: "Minority partition detected",
        detail: `${minority.length} nodes on side ${minoritySide.toUpperCase()} · read-only`,
      });
    }
    await wait(400);
    setBusy(false);
  }, [busy, nodes, emit]);

  const healPartition = useCallback(async () => {
    if (busy) return;
    setBusy(true);
    setStatus("recovering");
    emit({ kind: "info", label: "Reconnecting peers", detail: "Handshake in progress" });
    await wait(700);
    setPartition(false);
    emit({ kind: "info", label: "Log reconciliation", detail: "Catch-up AppendEntries" });
    await wait(900);
    emit({ kind: "ok", label: "Cluster reconverged", detail: `Term ${term} · 5/5 quorum` });
    setStatus("healthy");
    setBusy(false);
  }, [busy, emit, term]);

  const statusMeta = useMemo(() => {
    switch (status) {
      case "electing":
        return { color: "var(--warning)", label: "Election in progress" };
      case "partitioned":
        return { color: "var(--destructive)", label: "Partitioned" };
      case "recovering":
        return { color: "var(--primary)", label: "Recovering" };
      default:
        return { color: "var(--success)", label: "Healthy" };
    }
  }, [status]);

  return (
    <div className="card-elevated overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-border p-5">
        <div>
          <div className="text-display text-[15px] font-semibold tracking-tight">Failure Simulation Lab</div>
          <div className="mt-0.5 text-[12px] text-muted-foreground">
            Cinematic drills for leader loss and network partition
          </div>
        </div>
        <div className="flex items-center gap-2 rounded-full border border-border bg-surface px-3 py-1.5">
          <motion.span
            key={statusMeta.label}
            initial={{ scale: 0.6, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            className="h-1.5 w-1.5 rounded-full breathe"
            style={{ background: statusMeta.color }}
          />
          <span className="text-[11.5px] font-medium tracking-tight text-foreground/85">
            {statusMeta.label}
          </span>
          <span className="ml-1 font-mono text-[11px] tabular-nums text-muted-foreground">term {term}</span>
        </div>
      </div>

      <div className="grid gap-0 md:grid-cols-[minmax(0,1fr)_320px]">
        <div className="relative flex items-center justify-center bg-surface p-6">
          <ClusterViz size={SIZE} nodes={nodes} partition={partition} />
          <AnimatePresence>
            {status === "electing" && (
              <motion.div
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0 }}
                className="glass pointer-events-none absolute bottom-4 left-1/2 -translate-x-1/2 rounded-full px-3 py-1.5 text-[11.5px] font-medium tracking-tight text-foreground/85"
              >
                Electing new leader…
              </motion.div>
            )}
            {status === "partitioned" && (
              <motion.div
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0 }}
                className="glass pointer-events-none absolute bottom-4 left-1/2 -translate-x-1/2 rounded-full px-3 py-1.5 text-[11.5px] font-medium tracking-tight text-foreground/85"
              >
                Split-brain contained · quorum on majority side
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        <div className="flex flex-col border-t border-border md:border-l md:border-t-0">
          <div className="grid grid-cols-2 gap-2 p-4">
            <SimButton
              onClick={crashLeader}
              disabled={busy || !nodes.some((n) => n.role === "leader")}
              icon={Zap}
              label="Crash leader"
              hint="Trigger re-election"
            />
            <SimButton
              onClick={partition ? healPartition : partitionNetwork}
              disabled={busy}
              icon={partition ? PlayCircle : Scissors}
              label={partition ? "Heal network" : "Partition network"}
              hint={partition ? "Reconnect peers" : "Split into A / B"}
              tone={partition ? "primary" : undefined}
            />
            <SimButton
              onClick={reset}
              disabled={busy}
              icon={RotateCcw}
              label="Reset cluster"
              hint="Restore steady state"
              className="col-span-2"
            />
          </div>

          <div className="border-t border-border px-4 py-3 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
            Event Timeline
          </div>
          <div className="flex-1 space-y-1 overflow-y-auto px-2 pb-3">
            <AnimatePresence initial={false}>
              {events.map((e) => (
                <motion.div
                  key={e.id}
                  layout
                  initial={{ opacity: 0, x: -8, height: 0 }}
                  animate={{ opacity: 1, x: 0, height: "auto" }}
                  exit={{ opacity: 0, x: 8 }}
                  transition={{ type: "spring", stiffness: 220, damping: 26 }}
                  className="flex items-start gap-2.5 rounded-lg px-3 py-2 hover:bg-accent/40"
                >
                  <span
                    className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full"
                    style={{ background: eventColor(e.kind) }}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="text-[12.5px] font-medium text-foreground">{e.label}</div>
                    {e.detail && (
                      <div className="truncate text-[11px] text-muted-foreground">{e.detail}</div>
                    )}
                  </div>
                  <div className="font-mono text-[10.5px] tabular-nums text-muted-foreground">
                    {new Date(e.t).toLocaleTimeString(undefined, { hour12: false })}
                  </div>
                </motion.div>
              ))}
            </AnimatePresence>
          </div>
        </div>
      </div>
    </div>
  );
}

function eventColor(kind: Event["kind"]) {
  switch (kind) {
    case "ok":
      return "var(--success)";
    case "warn":
      return "var(--warning)";
    case "err":
      return "var(--destructive)";
    default:
      return "var(--primary)";
  }
}

function SimButton({
  onClick,
  disabled,
  icon: Icon,
  label,
  hint,
  className = "",
  tone,
}: {
  onClick: () => void;
  disabled?: boolean;
  icon: typeof Zap;
  label: string;
  hint: string;
  className?: string;
  tone?: "primary";
}) {
  return (
    <motion.button
      whileHover={{ y: disabled ? 0 : -1 }}
      whileTap={{ scale: disabled ? 1 : 0.98 }}
      onClick={onClick}
      disabled={disabled}
      className={`group flex flex-col items-start gap-1 rounded-xl border border-border bg-surface-elevated/70 p-3 text-left transition-colors ${
        disabled ? "opacity-50" : "hover:border-foreground/20 hover:bg-surface-elevated"
      } ${tone === "primary" ? "border-primary/40" : ""} ${className}`}
    >
      <div className="flex items-center gap-1.5 text-[12.5px] font-medium text-foreground">
        <Icon className="h-3.5 w-3.5" strokeWidth={1.75} />
        {label}
      </div>
      <div className="text-[11px] text-muted-foreground">{hint}</div>
    </motion.button>
  );
}

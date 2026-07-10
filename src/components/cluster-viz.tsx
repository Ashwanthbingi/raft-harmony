import { motion } from "framer-motion";
import { useMemo } from "react";

interface Node {
  id: string;
  label: string;
  role: "leader" | "follower" | "candidate" | "down";
  x: number;
  y: number;
}

interface Props {
  size?: number;
  nodes?: Node[];
  compact?: boolean;
}

const defaultNodes = (r: number, cx: number, cy: number): Node[] => {
  const followers = 4;
  const arr: Node[] = [{ id: "n0", label: "Node 01", role: "leader", x: cx, y: cy }];
  for (let i = 0; i < followers; i++) {
    const a = (i / followers) * Math.PI * 2 - Math.PI / 2;
    arr.push({
      id: `n${i + 1}`,
      label: `Node 0${i + 2}`,
      role: "follower",
      x: cx + Math.cos(a) * r,
      y: cy + Math.sin(a) * r,
    });
  }
  return arr;
};

export function ClusterViz({ size = 560, nodes, compact = false }: Props) {
  const cx = size / 2;
  const cy = size / 2;
  const r = size * 0.32;
  const data = useMemo(() => nodes ?? defaultNodes(r, cx, cy), [nodes, r, cx, cy]);
  const leader = data.find((n) => n.role === "leader")!;
  const others = data.filter((n) => n.id !== leader.id);

  return (
    <div className="relative" style={{ width: size, height: size }}>
      {/* Ambient rings */}
      <svg className="absolute inset-0" width={size} height={size}>
        <defs>
          <radialGradient id="glow" cx="50%" cy="50%">
            <stop offset="0%" stopColor="var(--leader)" stopOpacity="0.18" />
            <stop offset="70%" stopColor="var(--leader)" stopOpacity="0" />
          </radialGradient>
          <linearGradient id="edge" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stopColor="var(--primary)" stopOpacity="0.15" />
            <stop offset="50%" stopColor="var(--primary)" stopOpacity="0.55" />
            <stop offset="100%" stopColor="var(--primary)" stopOpacity="0.15" />
          </linearGradient>
        </defs>
        <circle cx={cx} cy={cy} r={r * 1.6} fill="url(#glow)" />
        {[0, 1, 2].map((i) => (
          <circle
            key={i}
            cx={cx}
            cy={cy}
            r={r * (0.6 + i * 0.3)}
            fill="none"
            stroke="var(--border)"
            strokeDasharray="2 6"
            opacity={0.5}
          />
        ))}

        {/* Curved connection lines */}
        {others.map((n) => {
          const mx = (leader.x + n.x) / 2;
          const my = (leader.y + n.y) / 2;
          const nx = -(n.y - leader.y);
          const ny = n.x - leader.x;
          const len = Math.hypot(nx, ny) || 1;
          const cxp = mx + (nx / len) * 30;
          const cyp = my + (ny / len) * 30;
          const path = `M ${leader.x} ${leader.y} Q ${cxp} ${cyp} ${n.x} ${n.y}`;
          return (
            <g key={n.id}>
              <path d={path} fill="none" stroke="url(#edge)" strokeWidth={1.5} />
              <path d={path} fill="none" stroke="var(--primary)" strokeWidth={2} className="flow-line" opacity={0.7} />
            </g>
          );
        })}

        {/* Packets */}
        {others.map((n, i) => {
          const mx = (leader.x + n.x) / 2;
          const my = (leader.y + n.y) / 2;
          const nx = -(n.y - leader.y);
          const ny = n.x - leader.x;
          const len = Math.hypot(nx, ny) || 1;
          const cxp = mx + (nx / len) * 30;
          const cyp = my + (ny / len) * 30;
          const path = `M ${leader.x} ${leader.y} Q ${cxp} ${cyp} ${n.x} ${n.y}`;
          return (
            <circle key={`p-${n.id}`} r={3.5} fill="var(--primary)">
              <animateMotion dur={`${2.4 + i * 0.3}s`} repeatCount="indefinite" path={path} />
              <animate attributeName="opacity" values="0;1;1;0" dur={`${2.4 + i * 0.3}s`} repeatCount="indefinite" />
            </circle>
          );
        })}
      </svg>

      {/* Leader pulse */}
      <div
        className="absolute rounded-full pulse-ring"
        style={{
          left: leader.x - 44,
          top: leader.y - 44,
          width: 88,
          height: 88,
          background: "color-mix(in oklab, var(--leader) 25%, transparent)",
        }}
      />

      {/* Nodes */}
      {data.map((n, i) => (
        <motion.div
          key={n.id}
          className="absolute"
          initial={{ opacity: 0, scale: 0.6 }}
          animate={{ opacity: 1, scale: 1, y: [0, -4, 0] }}
          transition={{
            opacity: { duration: 0.6, delay: i * 0.08 },
            scale: { type: "spring", stiffness: 180, damping: 16, delay: i * 0.08 },
            y: { duration: 4 + i * 0.4, repeat: Infinity, ease: "easeInOut" },
          }}
          style={{
            left: n.x - (n.role === "leader" ? 44 : 36),
            top: n.y - (n.role === "leader" ? 44 : 36),
          }}
        >
          <NodeBadge role={n.role} label={n.label} compact={compact} />
        </motion.div>
      ))}
    </div>
  );
}

function NodeBadge({ role, label, compact }: { role: Node["role"]; label: string; compact: boolean }) {
  const isLeader = role === "leader";
  const size = isLeader ? 88 : 72;
  return (
    <div
      className="glass-strong flex flex-col items-center justify-center rounded-full text-center"
      style={{
        width: size,
        height: size,
        boxShadow: isLeader ? "var(--shadow-leader)" : "var(--shadow-card)",
      }}
    >
      <div className="flex items-center gap-1">
        <span
          className="inline-block h-1.5 w-1.5 rounded-full breathe"
          style={{ background: role === "down" ? "var(--destructive)" : "var(--success)" }}
        />
        <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
          {isLeader ? "Leader" : "Follower"}
        </span>
      </div>
      {!compact && <div className="mt-0.5 text-[11px] font-semibold text-foreground/90">{label}</div>}
    </div>
  );
}

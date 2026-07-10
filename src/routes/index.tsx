import { createFileRoute } from "@tanstack/react-router";
import { motion } from "framer-motion";
import { Activity, GitBranch, Layers, ShieldCheck, Zap, Network, Cpu, ArrowRight } from "lucide-react";
import { SiteNav } from "@/components/site-nav";
import { ClusterViz } from "@/components/cluster-viz";
import { Link } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
  head: () => ({
    meta: [
      { title: "Raftline — Distributed Key-Value Store" },
      { name: "description", content: "A fault-tolerant distributed key-value store built on the Raft consensus algorithm. Designed with precision." },
      { property: "og:title", content: "Raftline — Distributed Key-Value Store" },
      { property: "og:description", content: "Consensus, replication, and observability — engineered with clarity." },
    ],
  }),
  component: Landing,
});

const container = {
  hidden: {},
  show: { transition: { staggerChildren: 0.08, delayChildren: 0.2 } },
};
const item = {
  hidden: { opacity: 0, y: 16, filter: "blur(8px)" },
  show: { opacity: 1, y: 0, filter: "blur(0px)", transition: { type: "spring" as const, stiffness: 120, damping: 20 } },
};

function Landing() {
  return (
    <div className="relative min-h-screen overflow-hidden bg-background">
      <SiteNav />

      {/* Ambient mesh */}
      <div className="pointer-events-none absolute inset-0 mesh-bg mesh-anim opacity-70" />
      <div className="pointer-events-none absolute inset-x-0 top-0 h-[80vh] bg-[radial-gradient(60%_50%_at_50%_0%,color-mix(in_oklab,var(--primary)_10%,transparent),transparent_70%)]" />

      {/* Hero */}
      <section className="relative mx-auto flex min-h-screen max-w-7xl flex-col items-center justify-center px-6 pt-32">
        <motion.div variants={container} initial="hidden" animate="show" className="text-center">
          <motion.div variants={item} className="mx-auto mb-6 inline-flex items-center gap-2 rounded-full border border-border bg-surface-elevated/60 px-3 py-1 text-[12px] text-muted-foreground backdrop-blur">
            <span className="h-1.5 w-1.5 rounded-full bg-success breathe" />
            Cluster healthy · 5 nodes · Term 42
          </motion.div>
          <motion.h1 variants={item} className="text-display mx-auto max-w-4xl text-5xl font-semibold leading-[1.05] tracking-tight text-foreground sm:text-6xl md:text-7xl">
            Consensus, quietly perfected.
          </motion.h1>
          <motion.p variants={item} className="mx-auto mt-5 max-w-xl text-[17px] leading-relaxed text-muted-foreground">
            A distributed, fault-tolerant key-value store built on Raft. Elegant observability for engineers who care about the details.
          </motion.p>
          <motion.div variants={item} className="mt-8 flex items-center justify-center gap-3">
            <Link
              to="/dashboard"
              className="group inline-flex items-center gap-1.5 rounded-full bg-foreground px-5 py-2.5 text-[13.5px] font-medium text-background shadow-lg shadow-foreground/10 transition-transform hover:scale-[1.02] active:scale-[0.98]"
            >
              Open the console
              <ArrowRight className="h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5" />
            </Link>
            <a
              href="#architecture"
              className="inline-flex items-center gap-1.5 rounded-full border border-border bg-surface-elevated/60 px-5 py-2.5 text-[13.5px] font-medium text-foreground backdrop-blur transition-colors hover:bg-accent"
            >
              How it works
            </a>
          </motion.div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 40, scale: 0.94 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ delay: 0.6, type: "spring", stiffness: 90, damping: 18 }}
          className="mt-16 flex items-center justify-center"
        >
          <ClusterViz size={560} />
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 1.2, duration: 0.6 }}
          className="mt-10 grid w-full max-w-3xl grid-cols-3 gap-4 text-center"
        >
          {[
            ["99.999%", "Availability"],
            ["4.2ms", "Commit latency"],
            ["1.8M", "Ops / min"],
          ].map(([v, l]) => (
            <div key={l} className="card-elevated px-4 py-4">
              <div className="text-display text-2xl font-semibold tracking-tight">{v}</div>
              <div className="mt-0.5 text-[12px] text-muted-foreground">{l}</div>
            </div>
          ))}
        </motion.div>
      </section>

      {/* Feature sections */}
      <section id="architecture" className="relative mx-auto max-w-7xl px-6 py-32">
        <SectionHeader eyebrow="Architecture" title="Engineered for calm under load." />
        <div className="mt-14 grid gap-4 md:grid-cols-3">
          {features.map((f, i) => (
            <FeatureCard key={f.title} {...f} delay={i * 0.08} />
          ))}
        </div>
      </section>

      <section className="relative mx-auto max-w-7xl px-6 pb-32">
        <div className="card-elevated grid gap-10 overflow-hidden p-10 md:grid-cols-2 md:p-14">
          <div>
            <div className="text-[12px] font-medium uppercase tracking-[0.14em] text-primary">Consensus Flow</div>
            <h3 className="text-display mt-3 text-3xl font-semibold tracking-tight md:text-4xl">
              Leader election, made observable.
            </h3>
            <p className="mt-4 max-w-md text-[15px] leading-relaxed text-muted-foreground">
              Watch votes propagate, terms advance, and log entries replicate in real time. Every state
              transition is animated so operators build intuition, not anxiety.
            </p>
            <ul className="mt-6 space-y-3 text-[14px]">
              {[
                "Strong leader with continuous heartbeats",
                "Log replication with quorum acknowledgement",
                "Automatic failover in under 200ms",
                "Split-brain prevention via majority quorum",
              ].map((t) => (
                <li key={t} className="flex items-start gap-2.5 text-foreground/85">
                  <div className="mt-1.5 h-1 w-1 rounded-full bg-primary" />
                  {t}
                </li>
              ))}
            </ul>
          </div>
          <div className="relative flex items-center justify-center rounded-2xl bg-surface p-4">
            <ClusterViz size={420} compact />
          </div>
        </div>
      </section>

      <footer className="border-t border-border/70">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-8 text-[12.5px] text-muted-foreground">
          <div>© 2026 Raftline Systems</div>
          <div className="flex gap-6">
            <a className="hover:text-foreground" href="#">Docs</a>
            <a className="hover:text-foreground" href="#">Changelog</a>
            <a className="hover:text-foreground" href="#">Status</a>
          </div>
        </div>
      </footer>
    </div>
  );
}

function SectionHeader({ eyebrow, title }: { eyebrow: string; title: string }) {
  return (
    <div className="max-w-2xl">
      <div className="text-[12px] font-medium uppercase tracking-[0.14em] text-primary">{eyebrow}</div>
      <h2 className="text-display mt-3 text-4xl font-semibold tracking-tight md:text-5xl">{title}</h2>
    </div>
  );
}

const features = [
  { icon: ShieldCheck, title: "Fault Tolerance", desc: "Survives node failures and network partitions with automatic recovery." },
  { icon: GitBranch, title: "Raft Consensus", desc: "Strong consistency guarantees with a formally verified protocol." },
  { icon: Zap, title: "Low Latency", desc: "Single-digit millisecond commits within a datacenter." },
  { icon: Network, title: "Replication", desc: "Log-based replication with quorum acknowledgement." },
  { icon: Activity, title: "Observability", desc: "First-class metrics, tracing, and live topology view." },
  { icon: Layers, title: "Snapshotting", desc: "Compact state snapshots for fast recovery and restart." },
];

function FeatureCard({ icon: Icon, title, desc, delay }: { icon: typeof Activity; title: string; desc: string; delay: number }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-80px" }}
      transition={{ delay, type: "spring", stiffness: 100, damping: 20 }}
      whileHover={{ y: -3 }}
      className="card-elevated group p-6"
    >
      <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-accent">
        <Icon className="h-4.5 w-4.5 text-foreground" strokeWidth={1.75} />
      </div>
      <div className="text-display mt-5 text-[17px] font-semibold tracking-tight">{title}</div>
      <div className="mt-1.5 text-[13.5px] leading-relaxed text-muted-foreground">{desc}</div>
    </motion.div>
  );
}

// unused import guard
void Cpu;

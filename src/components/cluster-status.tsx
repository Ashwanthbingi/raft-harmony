import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { RefreshCw, AlertCircle, CheckCircle2, Circle } from "lucide-react";
import { raftAPI, ClusterStatus } from "@/lib/api";

export function ClusterStatusView() {
  const [status, setStatus] = useState<ClusterStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<number>(0);

  const fetchStatus = async () => {
    setLoading(true);
    const clusterStatus = await raftAPI.getClusterStatus();
    setStatus(clusterStatus);
    setLastUpdated(Date.now());
    setLoading(false);
  };

  useEffect(() => {
    fetchStatus();
    // Poll for status updates every 3 seconds
    const interval = setInterval(fetchStatus, 3000);
    return () => clearInterval(interval);
  }, []);

  const getStateColor = (state: string) => {
    switch (state) {
      case "Leader":
        return "bg-primary";
      case "Follower":
        return "bg-chart-3";
      case "Candidate":
        return "bg-warning";
      default:
        return "bg-muted-foreground";
    }
  };

  const getStateIcon = (state: string) => {
    switch (state) {
      case "Leader":
        return "👑";
      case "Follower":
        return "📍";
      case "Candidate":
        return "⏱️";
      default:
        return "❓";
    }
  };

  if (!status) {
    return (
      <div className="card-elevated p-6 animate-pulse">
        <div className="h-6 bg-accent rounded w-32 mb-4" />
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-12 bg-accent rounded" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ type: "spring", stiffness: 120, damping: 20 }}
      className="card-elevated overflow-hidden"
    >
      <div className="border-b border-border p-5 flex items-center justify-between">
        <div>
          <div className="text-display text-[15px] font-semibold tracking-tight">Cluster Status</div>
          <div className="mt-0.5 text-[12px] text-muted-foreground">
            {status.nodes.length} nodes • Last updated {Math.round((Date.now() - lastUpdated) / 1000)}s ago
          </div>
        </div>
        <button
          onClick={fetchStatus}
          disabled={loading}
          className="p-2 rounded-lg hover:bg-accent transition-colors disabled:opacity-50"
          title="Refresh cluster status"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
        </button>
      </div>

      <div className="divide-y divide-border">
        {status.nodes.map((node, i) => {
          const isLeader = node.isLeader;
          const isUp = true; // Simplified for now

          return (
            <motion.div
              key={node.id}
              initial={{ opacity: 0, x: -8 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: i * 0.05 }}
              className="p-4 hover:bg-accent/40 transition-colors"
            >
              <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3 min-w-0">
                  <div className={`h-8 w-8 rounded-lg ${getStateColor(node.state)} flex items-center justify-center text-lg shrink-0`}>
                    {getStateIcon(node.state)}
                  </div>
                  <div className="min-w-0">
                    <div className="text-[13px] font-medium text-foreground truncate">{node.id}</div>
                    <div className="text-[11px] text-muted-foreground truncate">
                      {node.address}:{node.httpPort}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-4 flex-shrink-0">
                  <div className="text-right">
                    <div className="flex items-center gap-2">
                      {isUp ? (
                        <CheckCircle2 className="h-3.5 w-3.5 text-success" />
                      ) : (
                        <AlertCircle className="h-3.5 w-3.5 text-destructive" />
                      )}
                      <span className="text-[12px] font-mono tabular-nums">Term {node.term}</span>
                    </div>
                    <div className="text-[11px] text-muted-foreground mt-1">
                      {node.state}
                      {isLeader && " • Leader"}
                    </div>
                  </div>

                  {isLeader && (
                    <motion.div
                      animate={{ scale: [1, 1.1, 1] }}
                      transition={{ duration: 2, repeat: Infinity }}
                      className="text-[20px]"
                      title="Current Leader"
                    >
                      ⭐
                    </motion.div>
                  )}
                </div>
              </div>

              {/* Status bar */}
              <div className="mt-3 flex gap-1">
                {[0, 1, 2, 3, 4].map((i) => (
                  <motion.div
                    key={i}
                    animate={{ height: [4, 8, 4] }}
                    transition={{ duration: 0.6, delay: i * 0.1, repeat: Infinity }}
                    className={`flex-1 rounded-full ${isUp ? "bg-success" : "bg-destructive"} opacity-60`}
                  />
                ))}
              </div>
            </motion.div>
          );
        })}
      </div>

      {/* Summary section */}
      <div className="bg-surface p-4 border-t border-border">
        <div className="grid grid-cols-3 gap-4">
          <div>
            <div className="text-[11px] text-muted-foreground font-medium uppercase tracking-wider">Total Nodes</div>
            <div className="text-[18px] font-semibold mt-1">{status.nodes.length}</div>
          </div>
          <div>
            <div className="text-[11px] text-muted-foreground font-medium uppercase tracking-wider">Leader</div>
            <div className="text-[18px] font-semibold mt-1 truncate">{status.leader?.id || "—"}</div>
          </div>
          <div>
            <div className="text-[11px] text-muted-foreground font-medium uppercase tracking-wider">Health</div>
            <div className="flex items-center gap-2 mt-1">
              <span className="h-2 w-2 rounded-full bg-success animate-pulse" />
              <span className="text-[13px] font-medium">Healthy</span>
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );
}

import { createFileRoute } from "@tanstack/react-router";
import { motion } from "framer-motion";
import { Link } from "@tanstack/react-router";
import { ArrowLeft, Database } from "lucide-react";
import { KVOperations } from "@/components/kv-operations";
import { ClusterStatusView } from "@/components/cluster-status";

export const Route = createFileRoute("/operations")({
  head: () => ({
    meta: [
      { title: "Operations — Raft Cluster" },
      { name: "description", content: "Key-value operations and cluster status monitoring" },
    ],
  }),
  component: Operations,
});

function Operations() {
  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        className="sticky top-0 z-50 border-b border-border bg-background/70 backdrop-blur-xl"
      >
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link
              to="/dashboard"
              className="p-2 hover:bg-accent rounded-lg transition-colors"
              title="Back to dashboard"
            >
              <ArrowLeft className="h-4 w-4" />
            </Link>
            <div className="flex items-center gap-2">
              <Database className="h-5 w-5 text-primary" />
              <div>
                <h1 className="text-lg font-semibold text-foreground">Operations</h1>
                <p className="text-xs text-muted-foreground">Key-value store interactions</p>
              </div>
            </div>
          </div>
        </div>
      </motion.div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Status Section */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="mb-8"
        >
          <h2 className="text-[13px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">
            Cluster
          </h2>
          <ClusterStatusView />
        </motion.div>

        {/* KV Operations Section */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          <h2 className="text-[13px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">
            Key-Value Operations
          </h2>
          <KVOperations />
        </motion.div>

        {/* Info Section */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          className="mt-8 card-elevated p-6 bg-surface-elevated/50"
        >
          <h3 className="text-[14px] font-semibold text-foreground mb-3">About This Interface</h3>
          <ul className="space-y-2 text-[13px] text-muted-foreground">
            <li className="flex items-start gap-2">
              <span className="text-primary mt-1">•</span>
              <span>
                <strong>Set Value:</strong> Submit key-value pairs to the Raft cluster. They will be replicated across all nodes.
              </span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-primary mt-1">•</span>
              <span>
                <strong>Get Value:</strong> Retrieve values from any node in the cluster. Reading is eventually consistent.
              </span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-primary mt-1">•</span>
              <span>
                <strong>Cluster Status:</strong> Monitor node health, roles (Leader/Follower), and terms in real-time.
              </span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-primary mt-1">•</span>
              <span>
                <strong>Default Nodes:</strong> localhost:8001 (Leader), localhost:8002, localhost:8003 (Followers)
              </span>
            </li>
          </ul>
        </motion.div>

        {/* Connection Guide */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
          className="mt-6 card-elevated p-6 border-2 border-dashed border-border"
        >
          <h3 className="text-[14px] font-semibold text-foreground mb-3">Getting Started</h3>
          <ol className="space-y-2 text-[12px] text-muted-foreground font-mono">
            <li>1. Start the Raft cluster:</li>
            <li className="ml-4 text-primary">go run cmd/node/main.go --id=node1 ...</li>
            <li className="mt-3">2. Check Cluster Status above - you should see 3 nodes</li>
            <li className="mt-3">3. Try setting a value: key="greeting", value="hello"</li>
            <li className="mt-3">4. Retrieve it: key="greeting"</li>
          </ol>
        </motion.div>
      </div>
    </div>
  );
}

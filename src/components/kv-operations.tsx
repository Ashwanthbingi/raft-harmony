import { useState } from "react";
import { motion } from "framer-motion";
import { Copy, Check, AlertCircle, Loader2 } from "lucide-react";
import { raftAPI, SetResponse, GetResponse } from "@/lib/api";

export function KVOperations() {
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const [getKey, setGetKey] = useState("");
  const [setResult, setSetResult] = useState<SetResponse | null>(null);
  const [getResult, setGetResult] = useState<GetResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleSet = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!key.trim() || !value.trim()) return;

    setLoading(true);
    const result = await raftAPI.set(key, value);
    setSetResult(result);
    setLoading(false);

    if (result.success) {
      setKey("");
      setValue("");
    }
  };

  const handleGet = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!getKey.trim()) return;

    setLoading(true);
    const result = await raftAPI.get(getKey);
    setGetResult(result);
    setLoading(false);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      {/* SET Operation */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ type: "spring", stiffness: 120, damping: 20 }}
        className="card-elevated p-6"
      >
        <div className="mb-4">
          <h3 className="text-display text-[15px] font-semibold tracking-tight">Set Key-Value</h3>
          <p className="mt-1 text-[12px] text-muted-foreground">Submit a new key-value pair to the cluster</p>
        </div>

        <form onSubmit={handleSet} className="space-y-4">
          <div>
            <label className="block text-[12px] font-medium text-muted-foreground mb-2">Key</label>
            <input
              type="text"
              value={key}
              onChange={(e) => setKey(e.target.value)}
              placeholder="e.g., user:1234"
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-[13px] outline-none focus:border-primary focus:ring-1 focus:ring-primary"
              disabled={loading}
            />
          </div>

          <div>
            <label className="block text-[12px] font-medium text-muted-foreground mb-2">Value</label>
            <textarea
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder="e.g., JSON data or any string"
              rows={3}
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-[13px] outline-none focus:border-primary focus:ring-1 focus:ring-primary"
              disabled={loading}
            />
          </div>

          <button
            type="submit"
            disabled={loading || !key.trim() || !value.trim()}
            className="w-full rounded-lg bg-primary px-4 py-2 text-[13px] font-medium text-background transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Submitting...
              </>
            ) : (
              "Set Value"
            )}
          </button>
        </form>

        {setResult && (
          <motion.div
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            className={`mt-4 rounded-lg p-3 flex gap-3 ${
              setResult.success ? "bg-success/10 text-success" : "bg-destructive/10 text-destructive"
            }`}
          >
            {setResult.success ? (
              <Check className="h-4 w-4 shrink-0 mt-0.5" />
            ) : (
              <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
            )}
            <div>
              <div className="text-[12px] font-medium">{setResult.success ? "Success" : "Error"}</div>
              <div className="text-[11px] opacity-90 mt-0.5">{setResult.message}</div>
            </div>
          </motion.div>
        )}
      </motion.div>

      {/* GET Operation */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ type: "spring", stiffness: 120, damping: 20, delay: 0.1 }}
        className="card-elevated p-6"
      >
        <div className="mb-4">
          <h3 className="text-display text-[15px] font-semibold tracking-tight">Get Key Value</h3>
          <p className="mt-1 text-[12px] text-muted-foreground">Retrieve a value from any node in the cluster</p>
        </div>

        <form onSubmit={handleGet} className="space-y-4">
          <div>
            <label className="block text-[12px] font-medium text-muted-foreground mb-2">Key</label>
            <input
              type="text"
              value={getKey}
              onChange={(e) => setGetKey(e.target.value)}
              placeholder="e.g., user:1234"
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-[13px] outline-none focus:border-primary focus:ring-1 focus:ring-primary"
              disabled={loading}
            />
          </div>

          <button
            type="submit"
            disabled={loading || !getKey.trim()}
            className="w-full rounded-lg bg-primary px-4 py-2 text-[13px] font-medium text-background transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Fetching...
              </>
            ) : (
              "Get Value"
            )}
          </button>
        </form>

        {getResult && (
          <motion.div
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            className={`mt-4 rounded-lg p-3 flex gap-3 ${
              getResult.success ? "bg-success/10" : "bg-destructive/10"
            }`}
          >
            {getResult.success ? (
              <Check className="h-4 w-4 shrink-0 mt-0.5 text-success" />
            ) : (
              <AlertCircle className="h-4 w-4 shrink-0 mt-0.5 text-destructive" />
            )}
            <div className="flex-1">
              <div className={`text-[12px] font-medium ${getResult.success ? "text-success" : "text-destructive"}`}>
                {getResult.success ? "Found" : "Not Found"}
              </div>
              {getResult.success && getResult.value ? (
                <div className="mt-2 rounded bg-surface p-2 font-mono text-[11px] break-all relative group">
                  {getResult.value}
                  <button
                    onClick={() => copyToClipboard(getResult.value!)}
                    className="absolute right-2 top-2 p-1 opacity-0 group-hover:opacity-100 transition-opacity bg-surface-elevated rounded"
                    title="Copy value"
                  >
                    {copied ? (
                      <Check className="h-3 w-3 text-success" />
                    ) : (
                      <Copy className="h-3 w-3 text-muted-foreground hover:text-foreground" />
                    )}
                  </button>
                </div>
              ) : (
                <div className="text-[11px] opacity-90 mt-0.5 text-destructive">{getResult.error}</div>
              )}
            </div>
          </motion.div>
        )}
      </motion.div>
    </div>
  );
}

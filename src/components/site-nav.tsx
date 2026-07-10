import { Link } from "@tanstack/react-router";
import { motion } from "framer-motion";

export function SiteNav() {
  return (
    <motion.header
      initial={{ y: -20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ type: "spring", stiffness: 140, damping: 20 }}
      className="fixed left-1/2 top-4 z-50 -translate-x-1/2"
    >
      <nav className="glass flex items-center gap-1 rounded-full px-2 py-2 shadow-[0_10px_40px_-20px_rgba(15,23,42,0.25)]">
        <Link to="/" className="flex items-center gap-2 px-3 py-1.5">
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-foreground">
            <div className="h-2.5 w-2.5 rounded-sm bg-background" />
          </div>
          <span className="text-display text-[13px] font-semibold tracking-tight">Raftline</span>
        </Link>
        <div className="mx-1 h-5 w-px bg-border" />
        <NavLink to="/">Overview</NavLink>
        <NavLink to="/dashboard">Dashboard</NavLink>
        <NavLink to="/operations">Operations</NavLink>
        <NavLink to="/logs">Logs</NavLink>
        <Link
          to="/dashboard"
          className="ml-1 rounded-full bg-foreground px-3.5 py-1.5 text-[12.5px] font-medium text-background transition-transform hover:scale-[1.02] active:scale-[0.98]"
        >
          Open Console
        </Link>
      </nav>
    </motion.header>
  );
}

function NavLink({ to, children }: { to: string; children: React.ReactNode }) {
  return (
    <Link
      to={to}
      className="rounded-full px-3 py-1.5 text-[12.5px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      activeProps={{ className: "!text-foreground !bg-accent" }}
      activeOptions={{ exact: true }}
    >
      {children}
    </Link>
  );
}

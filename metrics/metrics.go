package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// RaftMetrics holds all Prometheus metrics for Raft consensus
type RaftMetrics struct {
	// State metrics
	CurrentTerm prometheus.Gauge
	LeaderID    prometheus.Gauge
	RaftState   prometheus.GaugeVec

	// Log metrics
	LogSize              prometheus.Gauge
	LastAppliedIndex     prometheus.Gauge
	CommitIndex          prometheus.Gauge
	LastIncludedIndex    prometheus.Gauge
	ReplicationLag       prometheus.GaugeVec
	LogEntriesReplicated prometheus.CounterVec

	// Election metrics
	ElectionsTotal    prometheus.Counter
	ElectionDuration  prometheus.Histogram
	TermChanges       prometheus.Counter
	LeaderChanges     prometheus.Counter

	// RPC metrics
	AppendEntriesTotal   prometheus.CounterVec
	AppendEntriesLatency prometheus.HistogramVec
	AppendEntriesFailed  prometheus.CounterVec
	RequestVoteTotal     prometheus.CounterVec
	RequestVoteLatency   prometheus.HistogramVec
	RequestVoteFailed    prometheus.CounterVec
	InstallSnapshotTotal prometheus.CounterVec

	// Snapshot metrics
	SnapshotGenerated     prometheus.Counter
	SnapshotSize          prometheus.Histogram
	SnapshotDuration      prometheus.Histogram
	SnapshotInstalled     prometheus.Counter
	LogEntriesTruncated   prometheus.Counter

	// KV Store metrics
	KVOperationsTotal prometheus.CounterVec
	KVLatency         prometheus.HistogramVec
}

// NewRaftMetrics creates and registers all Raft metrics
func NewRaftMetrics(nodeID string) *RaftMetrics {
	m := &RaftMetrics{
		// State metrics
		CurrentTerm: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "raft_current_term",
			Help:        "Current term of the Raft node",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		LeaderID: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "raft_leader_id",
			Help:        "ID of the current leader (1 if leader, 0 otherwise)",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		RaftState: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "raft_state",
				Help:        "Current state of the Raft node (1=Follower, 2=Candidate, 3=Leader)",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			[]string{"state"},
		),

		// Log metrics
		LogSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "raft_log_size",
			Help:        "Total log size (snapshot + in-memory entries)",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		LastAppliedIndex: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "raft_last_applied_index",
			Help:        "Index of the last applied log entry",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		CommitIndex: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "raft_commit_index",
			Help:        "Index of the commit index",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		LastIncludedIndex: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "raft_last_included_index",
			Help:        "Index of the last entry included in snapshot",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		ReplicationLag: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "raft_replication_lag",
				Help:        "Replication lag for each follower (entries behind)",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			[]string{"peer_id"},
		),

		LogEntriesReplicated: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "raft_log_entries_replicated_total",
				Help:        "Total log entries replicated to peers",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			[]string{"peer_id"},
		),

		// Election metrics
		ElectionsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "raft_elections_total",
			Help:        "Total number of elections conducted",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		ElectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:        "raft_election_duration_seconds",
			Help:        "Duration of election process in seconds",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
			Buckets:     []float64{.01, .05, .1, .5, 1, 2, 5},
		}),

		TermChanges: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "raft_term_changes_total",
			Help:        "Total number of term changes",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		LeaderChanges: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "raft_leader_changes_total",
			Help:        "Total number of leader changes",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		// RPC metrics
		AppendEntriesTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "raft_append_entries_total",
				Help:        "Total AppendEntries RPC calls",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			[]string{"peer_id"},
		),

		AppendEntriesLatency: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "raft_append_entries_latency_seconds",
				Help:        "AppendEntries RPC latency in seconds",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
				Buckets:     []float64{.001, .01, .05, .1, .5, 1},
			},
			[]string{"peer_id"},
		),

		AppendEntriesFailed: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "raft_append_entries_failed_total",
				Help:        "Total failed AppendEntries RPC calls",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			[]string{"peer_id"},
		),

		RequestVoteTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "raft_request_vote_total",
				Help:        "Total RequestVote RPC calls",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			[]string{"peer_id"},
		),

		RequestVoteLatency: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "raft_request_vote_latency_seconds",
				Help:        "RequestVote RPC latency in seconds",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
				Buckets:     []float64{.001, .01, .05, .1, .5, 1},
			},
			[]string{"peer_id"},
		),

		RequestVoteFailed: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "raft_request_vote_failed_total",
				Help:        "Total failed RequestVote RPC calls",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			[]string{"peer_id"},
		),

		InstallSnapshotTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "raft_install_snapshot_total",
				Help:        "Total InstallSnapshot RPC calls",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			[]string{"peer_id"},
		),

		// Snapshot metrics
		SnapshotGenerated: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "raft_snapshots_generated_total",
			Help:        "Total snapshots generated",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		SnapshotSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:        "raft_snapshot_size_bytes",
			Help:        "Snapshot size in bytes",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
			Buckets: []float64{
				1024,           // 1 KB
				10 * 1024,      // 10 KB
				100 * 1024,     // 100 KB
				1024 * 1024,    // 1 MB
				10 * 1024 * 1024, // 10 MB
			},
		}),

		SnapshotDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:        "raft_snapshot_duration_seconds",
			Help:        "Snapshot generation duration in seconds",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
			Buckets:     []float64{.01, .05, .1, .5, 1, 5},
		}),

		SnapshotInstalled: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "raft_snapshots_installed_total",
			Help:        "Total snapshots installed",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		LogEntriesTruncated: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "raft_log_entries_truncated_total",
			Help:        "Total log entries truncated by snapshots",
			ConstLabels: prometheus.Labels{"node_id": nodeID},
		}),

		// KV Store metrics
		KVOperationsTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "kv_operations_total",
				Help:        "Total KV store operations",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			[]string{"operation"},
		),

		KVLatency: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "kv_operation_latency_seconds",
				Help:        "KV store operation latency in seconds",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
				Buckets:     []float64{.001, .01, .05, .1, .5},
			},
			[]string{"operation"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		m.CurrentTerm,
		m.LeaderID,
		&m.RaftState,
		m.LogSize,
		m.LastAppliedIndex,
		m.CommitIndex,
		m.LastIncludedIndex,
		&m.ReplicationLag,
		&m.LogEntriesReplicated,
		m.ElectionsTotal,
		m.ElectionDuration,
		m.TermChanges,
		m.LeaderChanges,
		&m.AppendEntriesTotal,
		&m.AppendEntriesLatency,
		&m.AppendEntriesFailed,
		&m.RequestVoteTotal,
		&m.RequestVoteLatency,
		&m.RequestVoteFailed,
		&m.InstallSnapshotTotal,
		m.SnapshotGenerated,
		m.SnapshotSize,
		m.SnapshotDuration,
		m.SnapshotInstalled,
		m.LogEntriesTruncated,
		&m.KVOperationsTotal,
		&m.KVLatency,
	)

	return m
}

// RecordGetMetrics records a GET operation in metrics
func (m *RaftMetrics) RecordGetMetrics(latency float64) {
	m.KVOperationsTotal.WithLabelValues("GET").Inc()
	m.KVLatency.WithLabelValues("GET").Observe(latency)
}

// RecordSetMetrics records a SET operation in metrics
func (m *RaftMetrics) RecordSetMetrics(latency float64) {
	m.KVOperationsTotal.WithLabelValues("SET").Inc()
	m.KVLatency.WithLabelValues("SET").Observe(latency)
}

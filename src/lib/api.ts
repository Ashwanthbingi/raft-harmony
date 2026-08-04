// API service for communicating with Raft backend
// Handles SET/GET operations and cluster status

export interface ClusterNode {
  id: string;
  address: string;
  httpPort: number;
  state: 'Leader' | 'Follower' | 'Candidate';
  term: number;
  isLeader: boolean;
  isUp?: boolean;
}

export interface SetResponse {
  success: boolean;
  message: string;
  index?: number;
  term?: number;
}

export interface GetResponse {
  success: boolean;
  value?: string;
  error?: string;
}

export interface ClusterStatus {
  nodes: ClusterNode[];
  leader: ClusterNode | null;
  lastUpdate: number;
}

// Default cluster nodes - can be changed or provided via VITE_RAFT_NODES env var
const DEFAULT_NODES: ClusterNode[] = [
  {
    id: 'node1',
    address: 'localhost',
    httpPort: 8001,
    state: 'Leader',
    term: 0,
    isLeader: true,
  },
  {
    id: 'node2',
    address: 'localhost',
    httpPort: 8002,
    state: 'Follower',
    term: 0,
    isLeader: false,
  },
  {
    id: 'node3',
    address: 'localhost',
    httpPort: 8003,
    state: 'Follower',
    term: 0,
    isLeader: false,
  },
];

function parseEnvNodes(): ClusterNode[] {
  try {
    const raw = (typeof import.meta !== 'undefined' && (import.meta as any).env && (import.meta as any).env.VITE_RAFT_NODES) || '';
    if (!raw) return DEFAULT_NODES;
    const parts = raw.split(',').map((s: string) => s.trim()).filter(Boolean);
    return parts.map((p: string, i: number) => {
      const [host, port] = p.split(":");
      return {
        id: `node${i + 1}`,
        address: host || 'localhost',
        httpPort: port ? parseInt(port, 10) : 8001 + i,
        state: 'Follower' as const,
        term: 0,
        isLeader: i === 0,
      };
    });
  } catch (e) {
    return DEFAULT_NODES;
  }
}

class RaftAPI {
  private nodes: ClusterNode[];
  private leaderIndex: number = 0;

  constructor(nodes: ClusterNode[] = parseEnvNodes()) {
    this.nodes = nodes;
  }

  getBaseURL(nodeId?: string): string {
    if (nodeId) {
      const node = this.nodes.find(n => n.id === nodeId);
      if (node) {
        return `http://${node.address}:${node.httpPort}`;
      }
    }
    // Use leader or first node
    const leader = this.nodes.find(n => n.isLeader);
    const targetNode = leader || this.nodes[0];
    return `http://${targetNode.address}:${targetNode.httpPort}`;
  }

  async set(key: string, value: string, nodeId?: string): Promise<SetResponse> {
    try {
      const baseURL = this.getBaseURL(nodeId);
      const response = await fetch(`${baseURL}/set`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ key, value }),
      });

      const text = await response.text();

      if (!response.ok) {
        return {
          success: false,
          message: text || `Failed to set value (Status: ${response.status})`,
        };
      }

      return {
        success: true,
        message: text || 'Value set successfully',
      };
    } catch (error) {
      return {
        success: false,
        message: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`,
      };
    }
  }

  async get(key: string, nodeId?: string): Promise<GetResponse> {
    try {
      const baseURL = this.getBaseURL(nodeId);
      const response = await fetch(`${baseURL}/get?key=${encodeURIComponent(key)}`);

      if (!response.ok) {
        return {
          success: false,
          error: `Key not found (Status: ${response.status})`,
        };
      }

      const value = await response.text();
      return {
        success: true,
        value,
      };
    } catch (error) {
      return {
        success: false,
        error: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`,
      };
    }
  }

  async checkNodeStatus(nodeId: string): Promise<{ isUp: boolean; info?: any }> {
    try {
      const baseURL = this.getBaseURL(nodeId);
      const response = await fetch(`${baseURL}/health`);
      if (!response.ok) return { isUp: false };
      const data = await response.json().catch(() => null);
      return { isUp: true, info: data };
    } catch {
      return { isUp: false };
    }
  }

  async getClusterStatus(): Promise<ClusterStatus> {
    const nodes = [...this.nodes];
    const statusPromises = nodes.map(async (node) => {
      const res = await this.checkNodeStatus(node.id);
      const updated: ClusterNode = { ...node };
      if (res.isUp && res.info) {
        // Try to derive authoritative info from /health
        const info = res.info as any;
        if (typeof info.isLeader === 'boolean') {
          updated.isLeader = info.isLeader;
        }
        if (typeof info.currentTerm === 'number') {
          updated.term = info.currentTerm;
        }
        // address/httpPort unchanged here
      }
      return { ...updated, isUp: res.isUp } as any;
    });

    const results = await Promise.all(statusPromises);
    const leader = results.find((n: any) => n.isLeader && n.isUp) || results[0];

    return {
      nodes: results as ClusterNode[],
      leader: (leader as ClusterNode) || null,
      lastUpdate: Date.now(),
    };
  }

  updateNodes(nodes: ClusterNode[]): void {
    this.nodes = nodes;
  }

  getNodes(): ClusterNode[] {
    return this.nodes;
  }
}

export const raftAPI = new RaftAPI();

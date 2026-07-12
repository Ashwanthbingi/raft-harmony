// API service for communicating with Raft backend
// Handles SET/GET operations and cluster status

export interface ClusterNode {
  id: string;
  address: string;
  httpPort: number;
  state: 'Leader' | 'Follower' | 'Candidate';
  term: number;
  isLeader: boolean;
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

// Parse VITE_RAFT_NODES environment variable to override default nodes
// Format: "http://host1:port1,http://host2:port2,http://host3:port3"
function parseRaftNodesEnv(): ClusterNode[] {
  const envNodes = import.meta.env.VITE_RAFT_NODES;
  if (!envNodes) {
    return DEFAULT_NODES;
  }

  try {
    return envNodes.split(',').map((url, index) => {
      const urlObj = new URL(url.trim());
      return {
        id: `node${index + 1}`,
        address: urlObj.hostname,
        httpPort: parseInt(urlObj.port || '8001', 10),
        state: index === 0 ? 'Leader' : 'Follower',
        term: 0,
        isLeader: index === 0,
      };
    });
  } catch {
    console.warn('Failed to parse VITE_RAFT_NODES, using defaults');
    return DEFAULT_NODES;
  }
}

// Default cluster nodes - can be changed via VITE_RAFT_NODES env var
export const DEFAULT_NODES: ClusterNode[] = [
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

class RaftAPI {
  private nodes: ClusterNode[];
  private leaderIndex: number = 0;

  constructor(nodes: ClusterNode[] = parseRaftNodesEnv()) {
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

  async checkNodeStatus(nodeId: string): Promise<boolean> {
    try {
      const baseURL = this.getBaseURL(nodeId);
      const response = await fetch(`${baseURL}/get?key=__healthcheck__`, {
        method: 'GET',
      });
      return true; // If we can reach the node, it's up
    } catch {
      return false;
    }
  }

  async getClusterStatus(): Promise<ClusterStatus> {
    const nodes = [...this.nodes];
    const statusPromises = nodes.map(async (node) => {
      const isUp = await this.checkNodeStatus(node.id);
      return { ...node, isUp };
    });

    const results = await Promise.all(statusPromises);
    const leader = results.find(n => n.isLeader && n.isUp) || results[0];

    return {
      nodes: results,
      leader: leader || null,
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

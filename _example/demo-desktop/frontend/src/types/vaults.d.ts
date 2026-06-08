export interface VaultEntry {
  id: string;
  lastOpenedAt: string;
  name: string;
  path: string;
}

export interface VaultContext {
  entry: VaultEntry;
  memoDir: string;
  rootDir: string;
  veloDir: string;
}

export interface VaultStatus {
  active: VaultContext | null;
  dataFileExists: boolean;
  dataPath: string;
  vaults: VaultEntry[];
}

export interface VaultOpenResult {
  active?: VaultContext;
  created?: boolean;
  existing?: boolean;
  registry?: unknown;
}

import type { MemoRecord, MemoVisibility } from "./memos";
import type { ProjectID, ProjectListPayload, ProjectRecord } from "./projects";

export interface MemoCreatePayload {
  content: string;
  projectId?: ProjectID | "";
  visibility: MemoVisibility | string;
}

export interface MemoPatch {
  archived?: boolean;
  content?: string;
  pinned?: boolean;
  projectId?: ProjectID | "";
  updatedAt?: string;
  visibility?: MemoVisibility | string;
}

export interface MemoDeleteOptions {
  cleanupAssets?: boolean;
}

export interface MemoDeleteResult {
  assetErrors?: string[];
  assetsDeleted?: number;
  assetsSkipped?: number;
  success?: boolean;
}

export interface LocalMemoPayload {
  memo: MemoRecord | null;
  memos: MemoRecord[];
}

export interface ProjectCreatePayload {
  color?: string;
  name: string;
}

export type LoadProjectsResult = ProjectListPayload;

export type CreateProjectResult = ProjectRecord;

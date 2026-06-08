import type { ProjectID } from "./projects";

export type MemoID = string;

export type MemoVisibility = "PRIVATE" | "PROTECTED" | "PUBLIC";

export interface VisibilityMeta {
  icon: "lock" | "shield" | "globe" | string;
  label: string;
}

export interface MemoRecord {
  archived: boolean;
  content: string;
  createdAt: string;
  id: MemoID;
  pinned: boolean;
  projectId: ProjectID | "";
  updatedAt: string;
  visibility: MemoVisibility | string;
}

export interface TaskLine {
  checked: boolean;
  text: string;
}

export interface TodoItem {
  checked: boolean;
  id: string;
  lineIndex: number;
  memo: MemoRecord;
  memoId: MemoID;
  projectId: ProjectID | "";
  sourceText: string;
  text: string;
}

export interface TodoStats {
  done: number;
  open: number;
  total: number;
}

export type MemoSelectorType = "line" | "invalid" | "unsupported";

export interface MemoSelector {
  end?: number;
  raw: string;
  start?: number;
  type: MemoSelectorType;
}

export interface ParsedMemoReference {
  alias: string;
  embed: boolean;
  line?: number;
  raw?: string;
  selector: MemoSelector | null;
  target: string;
}

export interface MemoReferenceEdge extends ParsedMemoReference {
  sourceId: MemoID;
  targetId: MemoID | "";
}

export interface MemoReferenceIndex {
  incoming: Map<MemoID, MemoReferenceEdge[]>;
  memoById: Map<MemoID, MemoRecord>;
  memoByTitleKey: Map<string, MemoID>;
  outgoing: Map<MemoID, MemoReferenceEdge[]>;
  unresolved: MemoReferenceEdge[];
}

export interface MemoRenderContext {
  depth?: number;
  index?: MemoReferenceIndex;
  maxDepth?: number;
  readonly?: boolean;
  sourceId?: MemoID;
  stack?: MemoID[];
}

export interface MemoHeading {
  level: number;
  text: string;
}

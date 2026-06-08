import type { MemoID, MemoRecord } from "./memos";
import type { ProjectID } from "./projects";

export type MemoResourceType = "link" | "file" | "image";

export type MemoReferenceSyntax = "markdown" | "image" | "raw";

export interface MemoResourceReference {
  id: string;
  label: string;
  lineIndex: number;
  memo: MemoRecord;
  memoId: MemoID;
  sourceText: string;
  syntax: MemoReferenceSyntax;
  type: MemoResourceType;
  url: string;
}

export interface MemoResourceStats {
  files: number;
  images: number;
  total: number;
}

export interface MemoReferenceSource {
  memo: MemoRecord;
  projectId?: ProjectID | "";
}

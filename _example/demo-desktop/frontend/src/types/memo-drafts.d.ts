import type { MemoID, MemoVisibility } from "./memos";
import type { ProjectID } from "./projects";

export type MemoDraftKind = "composer" | "memo-edit";

export interface MemoDraftRecord {
  baseUpdatedAt: string;
  content: string;
  id: string;
  kind: MemoDraftKind;
  memoId: MemoID | "";
  projectId: ProjectID | "";
  updatedAt: string;
  visibility: MemoVisibility | string;
}

export declare const COMPOSER_DRAFT_ID: string;

export declare function memoEditDraftId(memoId: MemoID | string): string;

export declare function normalizeMemoDraftPayload(draft: unknown): MemoDraftRecord | null;

export declare function loadMemoDraftsFromVault(): Promise<MemoDraftRecord[]>;

export declare function upsertMemoDraftInVault(draft: Partial<MemoDraftRecord>): Promise<MemoDraftRecord>;

export declare function deleteMemoDraftInVault(id: string): Promise<{ success: boolean }>;

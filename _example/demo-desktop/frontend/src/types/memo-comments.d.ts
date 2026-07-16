import type { MemoID } from "./memos";

export interface MemoCommentRecord {
  content: string;
  createdAt: string;
  id: string;
  memoId: MemoID;
  path: string;
  private?: boolean;
  references: string[];
  tags: string[];
  updatedAt: string;
  visibility: string;
}

export interface MemoCommentDeleteOptions {
  cleanupAssets?: boolean;
}

export declare const MEMO_COMMENTS_STORAGE_KEY: string;

export declare function normalizeMemoCommentPayload(comment: unknown): MemoCommentRecord | null;

export declare function loadMemoCommentsFromVault(memoId?: MemoID | string): Promise<MemoCommentRecord[]>;

export declare function createMemoCommentInVault(memoId: MemoID | string, content: string, visibility?: string, isPrivate?: boolean): Promise<MemoCommentRecord>;

export declare function updateMemoCommentInVault(id: string, patch: Partial<MemoCommentRecord>): Promise<MemoCommentRecord>;

export declare function deleteMemoCommentInVault(id: string, options?: MemoCommentDeleteOptions): Promise<{ success: boolean }>;

import type { ProjectID } from "./projects";

export type TaskID = string;

export type TaskStatus = "open" | "completed" | "cancelled" | "archived";

export type TaskPriority = "none" | "low" | "medium" | "high";

export interface TaskReminder {
  at?: string;
  base?: string;
  fired?: boolean;
  offsetMinutes?: number;
  type: string;
}

export interface TaskRepeatEnd {
  at?: string;
  count?: number;
  type?: string;
}

export interface TaskRepeat {
  end?: TaskRepeatEnd;
  frequency: string;
  interval?: number;
  weekdays?: string[];
}

export interface TaskSource {
  line?: number;
  memoId?: string;
  memoPath?: string;
  text?: string;
  type?: string;
}

export interface TaskLink {
  id?: string;
  label?: string;
  type: string;
  url?: string;
}

export interface TaskNoteRef {
  createdAt: string;
  memoId: string;
  role: string;
  sortOrder: number;
}

export interface TaskRecord {
  cancelledAt: string;
  completedAt: string;
  contexts: string[];
  createdAt: string;
  dueAt: string;
  id: TaskID;
  links: TaskLink[];
  listId: string;
  notes: string;
  noteRefs: TaskNoteRef[];
  parentId: TaskID | "";
  path: string;
  priority: TaskPriority | string;
  private?: boolean;
  projectId: ProjectID | "";
  reminders: TaskReminder[];
  repeat: TaskRepeat;
  source: TaskSource;
  schemaVersion: number;
  startAt: string;
  status: TaskStatus | string;
  subtaskIds: TaskID[];
  tags: string[];
  timezone: string;
  title: string;
  updatedAt: string;
  visibility: string;
}

export interface TaskSummary {
  contexts: string[];
  completedAt: string;
  createdAt: string;
  dueAt: string;
  id: TaskID;
  listId: string;
  noteCount: number;
  parentId: TaskID | "";
  path: string;
  priority: TaskPriority | string;
  private?: boolean;
  projectId: ProjectID | "";
  source: TaskSource;
  startAt: string;
  status: TaskStatus | string;
  subtaskCount: number;
  tags: string[];
  title: string;
  updatedAt: string;
  visibility: string;
}

export interface TaskIndexEntry extends TaskSummary {}

export interface TaskIndexFile {
  rebuiltAt: string;
  schemaVersion: number;
  tasks: Record<TaskID, TaskIndexEntry>;
}

export interface TaskListResult {
  index: TaskIndexFile | null;
  tasks: TaskSummary[];
}

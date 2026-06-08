export type ProjectID = string;

export type ProjectFilter = "all" | "unassigned" | ProjectID;

export interface ProjectRecord {
  archived: boolean;
  color: string;
  createdAt: string;
  id: ProjectID;
  name: string;
  sortOrder: number;
  updatedAt: string;
}

export interface ProjectListPayload {
  activeProjectId: ProjectID | "";
  projects: ProjectRecord[];
}

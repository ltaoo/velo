export type StorageProvider = "s3" | "aliyun-oss" | "qiniu" | "local" | "local-oss" | string;

export type LocalStorageRootMode = "vault" | "absolute";

export interface LocalStorageSettings {
  root: string;
  rootMode: LocalStorageRootMode;
}

export interface CloudStorageProfile {
  accessKeyId: string;
  bucket: string;
  enabled: boolean;
  endpoint: string;
  forcePathStyle: boolean;
  id: string;
  local: LocalStorageSettings | null;
  name: string;
  pathPrefix: string;
  provider: StorageProvider;
  publicBaseUrl: string;
  region: string;
  secretAccessKey: string;
  sessionToken: string;
  useSSL: boolean;
}

export interface CloudStorageSettings {
  activeStorageId: string;
  defaultsInitialized: boolean;
  storages: CloudStorageProfile[];
}

export interface AssetReference {
  key: string;
  storageId: string;
}

export interface UploadedAsset {
  key: string;
  name: string;
  publicUrl: string;
  ref: string;
  storageId: string;
  type: string;
  url: string;
}

export interface FileInfo {
  name?: string;
  type?: string;
  url?: string;
}

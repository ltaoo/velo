export interface NativeAPIResponse<T = unknown> {
  code: number;
  data?: T;
  msg?: string;
}

export interface NativeAPIOptions<TArgs = Record<string, unknown>> {
  args?: TArgs;
  headers?: Record<string, unknown[]>;
  method?: string;
}

export type NativeAPIInvoker = <TData = unknown, TArgs = Record<string, unknown>>(
  url: string,
  options?: NativeAPIOptions<TArgs>,
) => Promise<NativeAPIResponse<TData>>;

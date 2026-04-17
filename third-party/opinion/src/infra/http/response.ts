export interface ApiOk<T> {
  success: true;
  data: T;
}

export interface ApiErr {
  success: false;
  code: string;
  message: string;
}

export type ApiResult<T> = ApiOk<T> | ApiErr;

export function ok<T>(data: T): ApiOk<T> {
  return { success: true, data };
}

export function err(code: string, message: string): ApiErr {
  return { success: false, code, message };
}

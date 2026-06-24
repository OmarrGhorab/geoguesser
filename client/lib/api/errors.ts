export type ApiErrorDetail = {
  code: string;
  message: string;
  request_id?: string;
  fields?: Array<{
    name: string;
    code: string;
    message: string;
  }>;
};

export class ApiError extends Error {
  readonly status: number;
  readonly detail: ApiErrorDetail;

  constructor(status: number, detail: ApiErrorDetail) {
    super(detail.message);
    this.name = "ApiError";
    this.status = status;
    this.detail = detail;
  }
}

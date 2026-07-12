import { z } from "zod";

const fieldErrorSchema = z.object({
  name: z.string(),
  code: z.string(),
  message: z.string(),
});

const errorDetailSchema = z.object({
  code: z.string(),
  message: z.string(),
  request_id: z.string().optional(),
  fields: z.array(fieldErrorSchema).optional(),
});

const errorResponseSchema = z.object({
  error: errorDetailSchema,
});

export type ApiFieldError = z.infer<typeof fieldErrorSchema>;
export type ApiErrorDetail = z.infer<typeof errorDetailSchema>;

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields: ApiFieldError[];
  readonly requestId?: string;

  constructor(status: number, detail: ApiErrorDetail) {
    super(detail.message);
    this.name = "ApiError";
    this.status = status;
    this.code = detail.code;
    this.fields = detail.fields ?? [];
    this.requestId = detail.request_id;
  }
}

export function parseApiErrorBody(body: unknown, status: number): ApiError {
  const parsed = errorResponseSchema.safeParse(body);
  if (!parsed.success) {
    return new ApiError(status, {
      code: "internal_error",
      message: "An unexpected error occurred.",
    });
  }
  return new ApiError(status, parsed.data.error);
}

/** Map backend error codes to localization keys under Auth.Errors */
export function mapApiErrorToMessageKey(error: ApiError): string {
  switch (error.code) {
    case "conflict":
      return "emailAlreadyExists";
    case "unauthorized":
      return "invalidCredentials";
    case "rate_limited":
      return "rateLimited";
    case "validation_failed":
      if (error.message.toLowerCase().includes("otp")) {
        return "invalidOtp";
      }
      if (error.message.toLowerCase().includes("password")) {
        return "invalidPassword";
      }
      return "validationFailed";
    default:
      return "generic";
  }
}

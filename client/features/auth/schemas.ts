import { z } from "zod";

export const AUTH_FIELD_LIMITS = {
  emailMaxLength: 254,
  passwordMinLength: 12,
  passwordMaxLength: 256,
  displayNameMinLength: 2,
  displayNameMaxLength: 32,
  otpLength: 6,
} as const;

const emailSchema = z
  .string()
  .trim()
  .min(1, "emailRequired")
  .email("emailInvalid")
  .max(AUTH_FIELD_LIMITS.emailMaxLength, "emailInvalid");

const passwordSchema = z
  .string()
  .min(AUTH_FIELD_LIMITS.passwordMinLength, "passwordTooShort")
  .max(AUTH_FIELD_LIMITS.passwordMaxLength, "passwordTooLong");

const displayNameSchema = z
  .string()
  .trim()
  .min(AUTH_FIELD_LIMITS.displayNameMinLength, "displayNameLength")
  .max(AUTH_FIELD_LIMITS.displayNameMaxLength, "displayNameLength");

const otpSchema = z
  .string()
  .trim()
  .regex(/^\d{6}$/, "otpInvalid");

/** POST /auth/register body (+ client confirm_password). */
export const registerFormSchema = z
  .object({
    email: emailSchema,
    password: passwordSchema,
    display_name: displayNameSchema,
    confirm_password: z.string().min(1, "confirmPasswordRequired"),
  })
  .superRefine((value, ctx) => {
    if (value.password !== value.confirm_password) {
      ctx.addIssue({
        code: "custom",
        path: ["confirm_password"],
        message: "passwordMismatch",
      });
    }
  });

export const registerRequestSchema = registerFormSchema.transform(
  ({ email, password, display_name }) => ({
    email,
    password,
    display_name,
  }),
);

/** POST /auth/login body */
export const loginFormSchema = z.object({
  email: emailSchema,
  password: z
    .string()
    .min(1, "passwordRequired")
    .max(AUTH_FIELD_LIMITS.passwordMaxLength),
});

/** POST /auth/forgot-password body */
export const forgotPasswordFormSchema = z.object({
  email: emailSchema,
});

/** POST /auth/reset-password body (+ client confirm_password). */
export const resetPasswordFormSchema = z
  .object({
    email: emailSchema,
    otp: otpSchema,
    new_password: passwordSchema,
    confirm_password: z.string().min(1, "confirmPasswordRequired"),
  })
  .superRefine((value, ctx) => {
    if (value.new_password !== value.confirm_password) {
      ctx.addIssue({
        code: "custom",
        path: ["confirm_password"],
        message: "passwordMismatch",
      });
    }
  });

export const resetPasswordRequestSchema = resetPasswordFormSchema.transform(
  ({ email, otp, new_password }) => ({
    email,
    otp,
    new_password,
  }),
);

export const authResponseSchema = z.object({
  user: z.object({
    id: z.string().uuid(),
    email: emailSchema,
    display_name: z.string(),
    role: z.string(),
  }),
});

export type RegisterFormInput = z.infer<typeof registerFormSchema>;
export type LoginFormInput = z.infer<typeof loginFormSchema>;
export type ForgotPasswordFormInput = z.infer<typeof forgotPasswordFormSchema>;
export type ResetPasswordFormInput = z.infer<typeof resetPasswordFormSchema>;

export type RegisterRequest = z.output<typeof registerRequestSchema>;
export type LoginRequest = z.infer<typeof loginFormSchema>;
export type ForgotPasswordRequest = z.infer<typeof forgotPasswordFormSchema>;
export type ResetPasswordRequest = z.output<typeof resetPasswordRequestSchema>;

export type FieldErrors = Partial<Record<string, string[]>>;

export function zodFieldErrors(error: z.ZodError): FieldErrors {
  const fieldErrors: FieldErrors = {};
  for (const issue of error.issues) {
    const key = issue.path.join(".") || "_form";
    const list = fieldErrors[key] ?? [];
    list.push(issue.message);
    fieldErrors[key] = list;
  }
  return fieldErrors;
}

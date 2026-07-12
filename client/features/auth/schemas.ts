/**
 * Request bodies aligned with backend/internal/auth/dto.go and OpenAPI.
 * Field names must match JSON tags exactly when posting to the API.
 */

/** POST /auth/register */
export type RegisterRequest = {
  email: string;
  password: string;
  display_name: string;
};

/** POST /auth/login */
export type LoginRequest = {
  email: string;
  password: string;
};

/** POST /auth/forgot-password */
export type ForgotPasswordRequest = {
  email: string;
};

/** POST /auth/reset-password */
export type ResetPasswordRequest = {
  email: string;
  otp: string;
  new_password: string;
};

/** Constraints from OpenAPI / service validation */
export const AUTH_FIELD_LIMITS = {
  emailMaxLength: 254,
  passwordMinLength: 12,
  passwordMaxLength: 256,
  displayNameMinLength: 2,
  displayNameMaxLength: 32,
  otpLength: 6,
} as const;

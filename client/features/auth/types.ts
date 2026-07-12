import type { FieldErrors } from "@/features/auth/schemas";

export type AuthActionStatus = "idle" | "error" | "success";

export type AuthActionState = {
  status: AuthActionStatus;
  /** Localization keys under Auth.Errors / field error keys */
  formError?: string;
  fieldErrors?: FieldErrors;
};

export const initialAuthActionState: AuthActionState = {
  status: "idle",
};

export type AuthUserDto = {
  id: string;
  email: string;
  display_name: string;
  role: string;
};

export type AuthResponse = {
  user: AuthUserDto;
};

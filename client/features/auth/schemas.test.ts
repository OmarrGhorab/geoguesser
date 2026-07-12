import { describe, expect, it } from "vitest";
import {
  forgotPasswordFormSchema,
  loginFormSchema,
  registerFormSchema,
  registerRequestSchema,
  resetPasswordFormSchema,
  resetPasswordRequestSchema,
  zodFieldErrors,
} from "@/features/auth/schemas";

describe("registerFormSchema", () => {
  it("accepts a valid register payload", () => {
    const result = registerFormSchema.safeParse({
      email: "player@example.com",
      password: "super-secure-1",
      display_name: "Explorer",
      confirm_password: "super-secure-1",
    });
    expect(result.success).toBe(true);
  });

  it("rejects short passwords and mismatched confirmation", () => {
    const result = registerFormSchema.safeParse({
      email: "player@example.com",
      password: "short",
      display_name: "E",
      confirm_password: "different",
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = zodFieldErrors(result.error);
      expect(errors.password?.[0]).toBe("passwordTooShort");
      expect(errors.display_name?.[0]).toBe("displayNameLength");
      expect(errors.confirm_password?.[0]).toBe("passwordMismatch");
    }
  });

  it("strips confirm_password from API request body", () => {
    const result = registerRequestSchema.safeParse({
      email: "player@example.com",
      password: "super-secure-1",
      display_name: "Explorer",
      confirm_password: "super-secure-1",
    });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data).toEqual({
        email: "player@example.com",
        password: "super-secure-1",
        display_name: "Explorer",
      });
      expect("confirm_password" in result.data).toBe(false);
    }
  });
});

describe("loginFormSchema", () => {
  it("requires email and password", () => {
    const result = loginFormSchema.safeParse({ email: "", password: "" });
    expect(result.success).toBe(false);
  });
});

describe("forgotPasswordFormSchema", () => {
  it("requires a valid email", () => {
    expect(
      forgotPasswordFormSchema.safeParse({ email: "not-an-email" }).success,
    ).toBe(false);
    expect(
      forgotPasswordFormSchema.safeParse({ email: "ok@example.com" }).success,
    ).toBe(true);
  });
});

describe("resetPasswordFormSchema", () => {
  it("requires email, 6-digit otp, and matching new password", () => {
    const result = resetPasswordFormSchema.safeParse({
      email: "player@example.com",
      otp: "12345",
      new_password: "super-secure-1",
      confirm_password: "super-secure-1",
    });
    expect(result.success).toBe(false);

    const ok = resetPasswordRequestSchema.safeParse({
      email: "player@example.com",
      otp: "123456",
      new_password: "super-secure-1",
      confirm_password: "super-secure-1",
    });
    expect(ok.success).toBe(true);
    if (ok.success) {
      expect(ok.data).toEqual({
        email: "player@example.com",
        otp: "123456",
        new_password: "super-secure-1",
      });
    }
  });
});

"use server";

import { redirect } from "next/navigation";
import { z } from "zod";
import { apiJson } from "@/lib/api/client";
import { ApiError, mapApiErrorToMessageKey } from "@/lib/api/errors";
import {
  forgotPasswordFormSchema,
  authResponseSchema,
  loginFormSchema,
  registerRequestSchema,
  resetPasswordRequestSchema,
  zodFieldErrors,
} from "@/features/auth/schemas";
import type { AuthActionState } from "@/features/auth/types";
import { routing, type AppLocale } from "@/lib/i18n/routing";

function formString(formData: FormData, key: string): string {
  const value = formData.get(key);
  return typeof value === "string" ? value : "";
}

function resolveLocale(formData: FormData): AppLocale {
  const locale = formString(formData, "locale");
  return routing.locales.includes(locale as AppLocale)
    ? (locale as AppLocale)
    : routing.defaultLocale;
}

function fromApiError(error: unknown): AuthActionState {
  if (error instanceof ApiError) {
    return {
      status: "error",
      formError: mapApiErrorToMessageKey(error),
    };
  }

  return {
    status: "error",
    formError: "generic",
  };
}

export async function registerAction(
  _prev: AuthActionState,
  formData: FormData,
): Promise<AuthActionState> {
  const locale = resolveLocale(formData);
  const parsed = registerRequestSchema.safeParse({
    email: formString(formData, "email"),
    password: formString(formData, "password"),
    display_name: formString(formData, "display_name"),
    confirm_password: formString(formData, "confirm_password"),
  });

  if (!parsed.success) {
    return {
      status: "error",
      fieldErrors: zodFieldErrors(parsed.error),
    };
  }

  try {
    await apiJson("/auth/register", authResponseSchema, {
      method: "POST",
      body: parsed.data,
      forwardCookies: true,
    });
  } catch (error) {
    return fromApiError(error);
  }

  redirect(`/${locale}`);
}

export async function loginAction(
  _prev: AuthActionState,
  formData: FormData,
): Promise<AuthActionState> {
  const locale = resolveLocale(formData);
  const parsed = loginFormSchema.safeParse({
    email: formString(formData, "email"),
    password: formString(formData, "password"),
  });

  if (!parsed.success) {
    return {
      status: "error",
      fieldErrors: zodFieldErrors(parsed.error),
    };
  }

  try {
    await apiJson("/auth/login", authResponseSchema, {
      method: "POST",
      body: parsed.data,
      forwardCookies: true,
    });
  } catch (error) {
    return fromApiError(error);
  }

  redirect(`/${locale}`);
}

export async function forgotPasswordAction(
  _prev: AuthActionState,
  formData: FormData,
): Promise<AuthActionState> {
  try {
    const parsed = forgotPasswordFormSchema.safeParse({
      email: formString(formData, "email"),
    });

    if (!parsed.success) {
      return {
        status: "error",
        fieldErrors: zodFieldErrors(parsed.error),
      };
    }

    try {
      await apiJson("/auth/forgot-password", z.undefined(), {
        method: "POST",
        body: parsed.data,
      });
    } catch (error) {
      return fromApiError(error);
    }

    // Only claim success after a successful API response (no false positives).
    return {
      status: "success",
    };
  } catch {
    return { status: "error", formError: "generic" };
  }
}

export async function resetPasswordAction(
  _prev: AuthActionState,
  formData: FormData,
): Promise<AuthActionState> {
  try {
    const parsed = resetPasswordRequestSchema.safeParse({
      email: formString(formData, "email"),
      otp: formString(formData, "otp"),
      new_password: formString(formData, "new_password"),
      confirm_password: formString(formData, "confirm_password"),
    });

    if (!parsed.success) {
      return {
        status: "error",
        fieldErrors: zodFieldErrors(parsed.error),
      };
    }

    try {
      await apiJson("/auth/reset-password", z.undefined(), {
        method: "POST",
        body: parsed.data,
      });
    } catch (error) {
      return fromApiError(error);
    }

    return {
      status: "success",
    };
  } catch {
    return { status: "error", formError: "generic" };
  }
}

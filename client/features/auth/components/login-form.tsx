"use client";

import { useActionState } from "react";
import Link from "next/link";
import { loginAction } from "@/features/auth/actions";
import { firstFieldError } from "@/features/auth/error-messages";
import {
  initialAuthActionState,
  type AuthActionState,
} from "@/features/auth/types";
import type { AppLocale } from "@/lib/i18n/routing";
import {
  AuthDivider,
  AuthField,
  AuthFormError,
  AuthPanel,
  AuthPasswordField,
  AuthPrimaryButton,
  AuthSocialButtons,
  AuthTitle,
} from "@/features/auth/components/auth-ui";

export type LoginFormLabels = {
  titleLine1: string;
  titleLine2: string;
  titleAccent: string;
  continueWithGoogle: string;
  continueWithDiscord: string;
  or: string;
  email: string;
  emailPlaceholder: string;
  password: string;
  passwordPlaceholder: string;
  forgotPassword: string;
  logIn: string;
  loggingIn: string;
  noAccount: string;
  signUp: string;
  showPassword: string;
  hidePassword: string;
  errors: Record<string, string>;
};

type LoginFormProps = {
  labels: LoginFormLabels;
  locale: AppLocale;
  initialFormError?: string;
};

function tError(errors: Record<string, string>, key: string | undefined) {
  if (!key) return undefined;
  return errors[key] ?? errors.generic;
}

export function LoginForm({
  labels,
  locale,
  initialFormError,
}: LoginFormProps) {
  const initialState: AuthActionState = initialFormError
    ? { status: "error", formError: initialFormError }
    : initialAuthActionState;
  const [state, formAction, isPending] = useActionState(
    loginAction,
    initialState,
  );

  return (
    <AuthPanel>
      <AuthTitle
        line1={labels.titleLine1}
        line2={labels.titleLine2}
        accent={labels.titleAccent}
      />

      <AuthSocialButtons
        googleLabel={labels.continueWithGoogle}
        discordLabel={labels.continueWithDiscord}
        locale={locale}
      />

      <AuthDivider label={labels.or} />

      <form action={formAction} className="flex flex-col gap-5" noValidate>
        <input type="hidden" name="locale" value={locale} />

        <AuthFormError message={tError(labels.errors, state.formError)} />

        <AuthField
          id="login-email"
          name="email"
          type="email"
          label={labels.email}
          placeholder={labels.emailPlaceholder}
          autoComplete="email"
          maxLength={254}
          required
          disabled={isPending}
          error={tError(
            labels.errors,
            firstFieldError(state.fieldErrors, "email"),
          )}
        />

        <AuthPasswordField
          id="login-password"
          name="password"
          label={labels.password}
          placeholder={labels.passwordPlaceholder}
          autoComplete="current-password"
          maxLength={256}
          required
          disabled={isPending}
          showPasswordLabel={labels.showPassword}
          hidePasswordLabel={labels.hidePassword}
          error={tError(
            labels.errors,
            firstFieldError(state.fieldErrors, "password"),
          )}
          labelEnd={
            <Link
              href={`/${locale}/forgot-password`}
              className="text-muted-foreground hover:text-foreground text-[12px] font-medium transition-colors"
            >
              {labels.forgotPassword}
            </Link>
          }
        />

        <AuthPrimaryButton
          isPending={isPending}
          pendingLabel={labels.loggingIn}
        >
          {labels.logIn}
        </AuthPrimaryButton>
      </form>

      <p className="mt-6 text-[13px] text-neutral-400">
        {labels.noAccount}{" "}
        <Link
          href={`/${locale}/sign-up`}
          className="font-bold text-white hover:underline"
        >
          {labels.signUp}
        </Link>
      </p>
    </AuthPanel>
  );
}

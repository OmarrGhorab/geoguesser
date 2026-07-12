"use client";

import { useActionState } from "react";
import Link from "next/link";
import { registerAction } from "@/features/auth/actions";
import { firstFieldError } from "@/features/auth/error-messages";
import { initialAuthActionState } from "@/features/auth/types";
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

export type SignUpFormLabels = {
  titleLine1: string;
  titleLine2: string;
  titleAccent: string;
  continueWithGoogle: string;
  continueWithDiscord: string;
  or: string;
  displayName: string;
  displayNamePlaceholder: string;
  email: string;
  emailPlaceholder: string;
  password: string;
  passwordPlaceholder: string;
  confirmPassword: string;
  confirmPasswordPlaceholder: string;
  createAccount: string;
  creatingAccount: string;
  alreadyHaveAccount: string;
  logIn: string;
  showPassword: string;
  hidePassword: string;
  errors: Record<string, string>;
};

type SignUpFormProps = {
  labels: SignUpFormLabels;
  locale: AppLocale;
};

function tError(errors: Record<string, string>, key: string | undefined) {
  if (!key) return undefined;
  return errors[key] ?? errors.generic;
}

export function SignUpForm({ labels, locale }: SignUpFormProps) {
  const [state, formAction, isPending] = useActionState(
    registerAction,
    initialAuthActionState,
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
          id="sign-up-display-name"
          name="display_name"
          type="text"
          label={labels.displayName}
          placeholder={labels.displayNamePlaceholder}
          autoComplete="nickname"
          minLength={2}
          maxLength={32}
          required
          disabled={isPending}
          error={tError(
            labels.errors,
            firstFieldError(state.fieldErrors, "display_name"),
          )}
        />
        <AuthField
          id="sign-up-email"
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
          id="sign-up-password"
          name="password"
          label={labels.password}
          placeholder={labels.passwordPlaceholder}
          autoComplete="new-password"
          minLength={12}
          maxLength={256}
          required
          disabled={isPending}
          showPasswordLabel={labels.showPassword}
          hidePasswordLabel={labels.hidePassword}
          error={tError(
            labels.errors,
            firstFieldError(state.fieldErrors, "password"),
          )}
        />
        <AuthPasswordField
          id="sign-up-confirm-password"
          name="confirm_password"
          label={labels.confirmPassword}
          placeholder={labels.confirmPasswordPlaceholder}
          autoComplete="new-password"
          minLength={12}
          maxLength={256}
          required
          disabled={isPending}
          showPasswordLabel={labels.showPassword}
          hidePasswordLabel={labels.hidePassword}
          error={tError(
            labels.errors,
            firstFieldError(state.fieldErrors, "confirm_password"),
          )}
        />
        <AuthPrimaryButton
          isPending={isPending}
          pendingLabel={labels.creatingAccount}
        >
          {labels.createAccount}
        </AuthPrimaryButton>
      </form>

      <p className="mt-6 text-[13px] text-neutral-400">
        {labels.alreadyHaveAccount}{" "}
        <Link
          href={`/${locale}/login`}
          className="font-bold text-white hover:underline"
        >
          {labels.logIn}
        </Link>
      </p>
    </AuthPanel>
  );
}

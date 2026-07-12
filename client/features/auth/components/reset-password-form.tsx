"use client";

import { useActionState } from "react";
import Link from "next/link";
import { resetPasswordAction } from "@/features/auth/actions";
import { firstFieldError } from "@/features/auth/error-messages";
import { initialAuthActionState } from "@/features/auth/types";
import {
  AuthField,
  AuthFormError,
  AuthItem,
  AuthPanel,
  AuthPasswordField,
  AuthPrimaryButton,
  AuthTitle,
  authPrimaryButtonClassName,
} from "@/features/auth/components/auth-ui";
import { cn } from "@/lib/utils";

export type ResetPasswordFormLabels = {
  titleLine1: string;
  titleLine2: string;
  titleAccent: string;
  description: string;
  email: string;
  emailPlaceholder: string;
  otp: string;
  otpPlaceholder: string;
  newPassword: string;
  newPasswordPlaceholder: string;
  confirmPassword: string;
  confirmPasswordPlaceholder: string;
  submit: string;
  submitting: string;
  rememberPassword: string;
  logIn: string;
  successTitle: string;
  successDescription: string;
  backToLogin: string;
  showPassword: string;
  hidePassword: string;
  errors: Record<string, string>;
};

type ResetPasswordFormProps = {
  labels: ResetPasswordFormLabels;
  locale: string;
};

function tError(errors: Record<string, string>, key: string | undefined) {
  if (!key) return undefined;
  return errors[key] ?? errors.generic;
}

export function ResetPasswordForm({ labels, locale }: ResetPasswordFormProps) {
  const [state, formAction, isPending] = useActionState(
    resetPasswordAction,
    initialAuthActionState,
  );

  if (state.status === "success") {
    return (
      <AuthPanel>
        <AuthTitle
          line1={labels.successTitle}
          description={labels.successDescription}
        />
        <AuthItem>
          <Link
            href={`/${locale}/login`}
            className={cn(authPrimaryButtonClassName, "block text-center")}
          >
            {labels.backToLogin}
          </Link>
        </AuthItem>
      </AuthPanel>
    );
  }

  return (
    <AuthPanel>
      <AuthTitle
        line1={labels.titleLine1}
        line2={labels.titleLine2}
        accent={labels.titleAccent}
        description={labels.description}
      />

      <form action={formAction} className="flex flex-col gap-5" noValidate>
        <AuthFormError message={tError(labels.errors, state.formError)} />

        <AuthField
          id="reset-email"
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
        <AuthField
          id="reset-otp"
          name="otp"
          type="text"
          label={labels.otp}
          placeholder={labels.otpPlaceholder}
          autoComplete="one-time-code"
          inputMode="numeric"
          pattern="[0-9]{6}"
          minLength={6}
          maxLength={6}
          spellCheck={false}
          required
          disabled={isPending}
          error={tError(
            labels.errors,
            firstFieldError(state.fieldErrors, "otp"),
          )}
        />
        <AuthPasswordField
          id="reset-new-password"
          name="new_password"
          label={labels.newPassword}
          placeholder={labels.newPasswordPlaceholder}
          autoComplete="new-password"
          minLength={12}
          maxLength={256}
          required
          disabled={isPending}
          showPasswordLabel={labels.showPassword}
          hidePasswordLabel={labels.hidePassword}
          error={tError(
            labels.errors,
            firstFieldError(state.fieldErrors, "new_password"),
          )}
        />
        <AuthPasswordField
          id="reset-confirm-password"
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
          pendingLabel={labels.submitting}
        >
          {labels.submit}
        </AuthPrimaryButton>
      </form>

      <p className="mt-6 text-[13px] text-neutral-400">
        {labels.rememberPassword}{" "}
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

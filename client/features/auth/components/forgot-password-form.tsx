"use client";

import { useActionState } from "react";
import Link from "next/link";
import { forgotPasswordAction } from "@/features/auth/actions";
import { firstFieldError } from "@/features/auth/error-messages";
import { initialAuthActionState } from "@/features/auth/types";
import {
  AuthField,
  AuthFormError,
  AuthItem,
  AuthPanel,
  AuthPrimaryButton,
  AuthTitle,
  authPrimaryButtonClassName,
} from "@/features/auth/components/auth-ui";
import { cn } from "@/lib/utils";

export type ForgotPasswordFormLabels = {
  titleLine1: string;
  titleLine2: string;
  titleAccent: string;
  description: string;
  email: string;
  emailPlaceholder: string;
  submit: string;
  submitting: string;
  rememberPassword: string;
  logIn: string;
  successTitle: string;
  successDescription: string;
  continueToReset: string;
  errors: Record<string, string>;
};

type ForgotPasswordFormProps = {
  labels: ForgotPasswordFormLabels;
  locale: string;
};

function tError(errors: Record<string, string>, key: string | undefined) {
  if (!key) return undefined;
  return errors[key] ?? errors.generic;
}

export function ForgotPasswordForm({
  labels,
  locale,
}: ForgotPasswordFormProps) {
  const [state, formAction, isPending] = useActionState(
    forgotPasswordAction,
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
            href={`/${locale}/reset-password`}
            className={cn(authPrimaryButtonClassName, "block text-center")}
          >
            {labels.continueToReset}
          </Link>
        </AuthItem>
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
          id="forgot-email"
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

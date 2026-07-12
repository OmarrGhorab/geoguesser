"use client";

import { useState } from "react";
import {
  AuthField,
  AuthFooter,
  AuthItem,
  AuthPanel,
  AuthPrimaryButton,
  AuthTitle,
  authPrimaryButtonClassName,
} from "@/features/auth/components/auth-ui";
import { cn } from "@/lib/utils";

/** Matches backend `auth.ForgotPasswordRequest`: email */
export type ForgotPasswordFormLabels = {
  titleLine1: string;
  titleLine2: string;
  titleAccent: string;
  description: string;
  email: string;
  emailPlaceholder: string;
  submit: string;
  rememberPassword: string;
  logIn: string;
  successTitle: string;
  successDescription: string;
  continueToReset: string;
};

type ForgotPasswordFormProps = {
  labels: ForgotPasswordFormLabels;
  loginHref: string;
  resetPasswordHref: string;
};

export function ForgotPasswordForm({
  labels,
  loginHref,
  resetPasswordHref,
}: ForgotPasswordFormProps) {
  const [submittedEmail, setSubmittedEmail] = useState<string | null>(null);

  if (submittedEmail) {
    const resetHref = `${resetPasswordHref}?email=${encodeURIComponent(submittedEmail)}`;

    return (
      <AuthPanel>
        <AuthTitle
          line1={labels.successTitle}
          description={labels.successDescription}
        />
        <AuthItem>
          <a
            href={resetHref}
            className={cn(authPrimaryButtonClassName, "block text-center")}
          >
            {labels.continueToReset}
          </a>
        </AuthItem>
        <AuthFooter
          prompt={labels.rememberPassword}
          href={loginHref}
          linkLabel={labels.logIn}
        />
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

      <form
        className="flex flex-col gap-5"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          const formData = new FormData(event.currentTarget);
          const email = String(formData.get("email") ?? "").trim();
          setSubmittedEmail(email);
        }}
      >
        <AuthField
          id="forgot-email"
          name="email"
          type="email"
          label={labels.email}
          placeholder={labels.emailPlaceholder}
          autoComplete="email"
          maxLength={254}
          required
        />
        <AuthPrimaryButton>{labels.submit}</AuthPrimaryButton>
      </form>

      <AuthFooter
        prompt={labels.rememberPassword}
        href={loginHref}
        linkLabel={labels.logIn}
      />
    </AuthPanel>
  );
}

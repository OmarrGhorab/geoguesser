"use client";

import { useState } from "react";
import {
  AuthField,
  AuthFooter,
  AuthItem,
  AuthPanel,
  AuthPasswordField,
  AuthPrimaryButton,
  AuthTitle,
  authPrimaryButtonClassName,
} from "@/features/auth/components/auth-ui";
import { cn } from "@/lib/utils";

/** Matches backend `auth.ResetPasswordRequest`: email, otp, new_password */
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
  rememberPassword: string;
  logIn: string;
  successTitle: string;
  successDescription: string;
  backToLogin: string;
  showPassword: string;
  hidePassword: string;
  passwordMismatch: string;
};

type ResetPasswordFormProps = {
  labels: ResetPasswordFormLabels;
  loginHref: string;
  defaultEmail?: string;
};

export function ResetPasswordForm({
  labels,
  loginHref,
  defaultEmail,
}: ResetPasswordFormProps) {
  const [submitted, setSubmitted] = useState(false);

  if (submitted) {
    return (
      <AuthPanel>
        <AuthTitle
          line1={labels.successTitle}
          description={labels.successDescription}
        />
        <AuthItem>
          <a
            href={loginHref}
            className={cn(authPrimaryButtonClassName, "block text-center")}
          >
            {labels.backToLogin}
          </a>
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

      <form
        className="flex flex-col gap-5"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const formData = new FormData(form);
          const password = String(formData.get("new_password") ?? "");
          const confirmPassword = String(
            formData.get("confirm_password") ?? "",
          );

          const confirmInput = form.elements.namedItem(
            "confirm_password",
          ) as HTMLInputElement | null;

          if (password !== confirmPassword) {
            confirmInput?.setCustomValidity(labels.passwordMismatch);
            confirmInput?.reportValidity();
            return;
          }

          confirmInput?.setCustomValidity("");
          setSubmitted(true);
        }}
      >
        <AuthField
          id="reset-email"
          name="email"
          type="email"
          label={labels.email}
          placeholder={labels.emailPlaceholder}
          autoComplete="email"
          maxLength={254}
          defaultValue={defaultEmail}
          required
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
          showPasswordLabel={labels.showPassword}
          hidePasswordLabel={labels.hidePassword}
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
          showPasswordLabel={labels.showPassword}
          hidePasswordLabel={labels.hidePassword}
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

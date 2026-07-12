"use client";

import {
  AuthDivider,
  AuthField,
  AuthFooter,
  AuthPanel,
  AuthPasswordField,
  AuthPrimaryButton,
  AuthSocialButtons,
  AuthTitle,
} from "@/features/auth/components/auth-ui";

/** Backend register body: email, password, display_name. confirm_password is client-only. */
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
  alreadyHaveAccount: string;
  logIn: string;
  showPassword: string;
  hidePassword: string;
  passwordMismatch: string;
};

type SignUpFormProps = {
  labels: SignUpFormLabels;
  loginHref: string;
};

export function SignUpForm({ labels, loginHref }: SignUpFormProps) {
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
      />

      <AuthDivider label={labels.or} />

      <form
        className="flex flex-col gap-5"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const formData = new FormData(form);
          const password = String(formData.get("password") ?? "");
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
        }}
      >
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
          showPasswordLabel={labels.showPassword}
          hidePasswordLabel={labels.hidePassword}
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
          showPasswordLabel={labels.showPassword}
          hidePasswordLabel={labels.hidePassword}
        />
        <AuthPrimaryButton>{labels.createAccount}</AuthPrimaryButton>
      </form>

      <AuthFooter
        prompt={labels.alreadyHaveAccount}
        href={loginHref}
        linkLabel={labels.logIn}
      />
    </AuthPanel>
  );
}

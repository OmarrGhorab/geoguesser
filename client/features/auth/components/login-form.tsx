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

/** Matches backend `auth.LoginRequest`: email, password */
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
  noAccount: string;
  signUp: string;
  showPassword: string;
  hidePassword: string;
};

type LoginFormProps = {
  labels: LoginFormLabels;
  signUpHref: string;
  forgotPasswordHref: string;
};

export function LoginForm({
  labels,
  signUpHref,
  forgotPasswordHref,
}: LoginFormProps) {
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
        }}
      >
        <AuthField
          id="login-email"
          name="email"
          type="email"
          label={labels.email}
          placeholder={labels.emailPlaceholder}
          autoComplete="email"
          maxLength={254}
          required
        />

        <AuthPasswordField
          id="login-password"
          name="password"
          label={labels.password}
          placeholder={labels.passwordPlaceholder}
          autoComplete="current-password"
          maxLength={256}
          required
          showPasswordLabel={labels.showPassword}
          hidePasswordLabel={labels.hidePassword}
          labelEnd={
            <a
              href={forgotPasswordHref}
              className="text-[12px] font-medium text-neutral-400 transition-colors hover:text-white"
            >
              {labels.forgotPassword}
            </a>
          }
        />

        <AuthPrimaryButton>{labels.logIn}</AuthPrimaryButton>
      </form>

      <AuthFooter
        prompt={labels.noAccount}
        href={signUpHref}
        linkLabel={labels.signUp}
      />
    </AuthPanel>
  );
}

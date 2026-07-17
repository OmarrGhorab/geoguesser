import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { LoginForm } from "@/features/auth/components/login-form";
import { authErrorLabels } from "@/features/auth/labels";
import type { AppLocale } from "@/lib/i18n/routing";
import { siteUrl } from "@/lib/site";

type LoginPageProps = Readonly<{
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ oauth_error?: string; reason?: string }>;
}>;

export async function generateMetadata({
  params,
}: LoginPageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({
    locale: locale as AppLocale,
    namespace: "Auth.Login",
  });

  return {
    metadataBase: siteUrl,
    title: t("metaTitle"),
    description: t("metaDescription"),
  };
}

export default async function LoginPage({
  params,
  searchParams,
}: LoginPageProps) {
  const { locale } = await params;
  const { oauth_error: oauthError, reason } = await searchParams;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const [t, tErrors] = await Promise.all([
    getTranslations({ locale: appLocale, namespace: "Auth.Login" }),
    getTranslations({ locale: appLocale, namespace: "Auth.Errors" }),
  ]);
  return (
    <LoginForm
      locale={appLocale}
      initialFormError={
        oauthError
          ? "oauthFailed"
          : reason === "session_expired"
            ? "sessionExpired"
            : undefined
      }
      labels={{
        titleLine1: t("titleLine1"),
        titleLine2: t("titleLine2"),
        titleAccent: t("titleAccent"),
        continueWithGoogle: t("continueWithGoogle"),
        continueWithDiscord: t("continueWithDiscord"),
        or: t("or"),
        email: t("email"),
        emailPlaceholder: t("emailPlaceholder"),
        password: t("password"),
        passwordPlaceholder: t("passwordPlaceholder"),
        forgotPassword: t("forgotPassword"),
        logIn: t("logIn"),
        loggingIn: t("loggingIn"),
        noAccount: t("noAccount"),
        signUp: t("signUp"),
        showPassword: t("showPassword"),
        hidePassword: t("hidePassword"),
        errors: authErrorLabels(tErrors),
      }}
    />
  );
}

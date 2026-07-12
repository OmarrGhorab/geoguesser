import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { LoginForm } from "@/features/auth/components/login-form";
import type { AppLocale } from "@/lib/i18n/routing";
import { siteUrl } from "@/lib/site";

type LoginPageProps = Readonly<{
  params: Promise<{ locale: string }>;
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

export default async function LoginPage({ params }: LoginPageProps) {
  const { locale } = await params;
  setRequestLocale(locale as AppLocale);

  const t = await getTranslations({
    locale: locale as AppLocale,
    namespace: "Auth.Login",
  });

  return (
    <LoginForm
      signUpHref={`/${locale}/sign-up`}
      forgotPasswordHref={`/${locale}/forgot-password`}
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
        noAccount: t("noAccount"),
        signUp: t("signUp"),
        showPassword: t("showPassword"),
        hidePassword: t("hidePassword"),
      }}
    />
  );
}

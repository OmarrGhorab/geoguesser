import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { SignUpForm } from "@/features/auth/components/sign-up-form";
import type { AppLocale } from "@/lib/i18n/routing";
import { siteUrl } from "@/lib/site";

type SignUpPageProps = Readonly<{
  params: Promise<{ locale: string }>;
}>;

export async function generateMetadata({
  params,
}: SignUpPageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({
    locale: locale as AppLocale,
    namespace: "Auth.SignUp",
  });

  return {
    metadataBase: siteUrl,
    title: t("metaTitle"),
    description: t("metaDescription"),
  };
}

export default async function SignUpPage({ params }: SignUpPageProps) {
  const { locale } = await params;
  setRequestLocale(locale as AppLocale);

  const t = await getTranslations({
    locale: locale as AppLocale,
    namespace: "Auth.SignUp",
  });

  return (
    <SignUpForm
      loginHref={`/${locale}/login`}
      labels={{
        titleLine1: t("titleLine1"),
        titleLine2: t("titleLine2"),
        titleAccent: t("titleAccent"),
        continueWithGoogle: t("continueWithGoogle"),
        continueWithDiscord: t("continueWithDiscord"),
        or: t("or"),
        displayName: t("displayName"),
        displayNamePlaceholder: t("displayNamePlaceholder"),
        email: t("email"),
        emailPlaceholder: t("emailPlaceholder"),
        password: t("password"),
        passwordPlaceholder: t("passwordPlaceholder"),
        confirmPassword: t("confirmPassword"),
        confirmPasswordPlaceholder: t("confirmPasswordPlaceholder"),
        createAccount: t("createAccount"),
        alreadyHaveAccount: t("alreadyHaveAccount"),
        logIn: t("logIn"),
        showPassword: t("showPassword"),
        hidePassword: t("hidePassword"),
        passwordMismatch: t("passwordMismatch"),
      }}
    />
  );
}

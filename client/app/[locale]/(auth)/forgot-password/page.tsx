import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { ForgotPasswordForm } from "@/features/auth/components/forgot-password-form";
import type { AppLocale } from "@/lib/i18n/routing";
import { siteUrl } from "@/lib/site";

type ForgotPasswordPageProps = Readonly<{
  params: Promise<{ locale: string }>;
}>;

export async function generateMetadata({
  params,
}: ForgotPasswordPageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({
    locale: locale as AppLocale,
    namespace: "Auth.ForgotPassword",
  });

  return {
    metadataBase: siteUrl,
    title: t("metaTitle"),
    description: t("metaDescription"),
  };
}

export default async function ForgotPasswordPage({
  params,
}: ForgotPasswordPageProps) {
  const { locale } = await params;
  setRequestLocale(locale as AppLocale);

  const t = await getTranslations({
    locale: locale as AppLocale,
    namespace: "Auth.ForgotPassword",
  });

  return (
    <ForgotPasswordForm
      loginHref={`/${locale}/login`}
      resetPasswordHref={`/${locale}/reset-password`}
      labels={{
        titleLine1: t("titleLine1"),
        titleLine2: t("titleLine2"),
        titleAccent: t("titleAccent"),
        description: t("description"),
        email: t("email"),
        emailPlaceholder: t("emailPlaceholder"),
        submit: t("submit"),
        rememberPassword: t("rememberPassword"),
        logIn: t("logIn"),
        successTitle: t("successTitle"),
        successDescription: t("successDescription"),
        continueToReset: t("continueToReset"),
      }}
    />
  );
}

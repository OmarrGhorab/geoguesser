import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { ResetPasswordForm } from "@/features/auth/components/reset-password-form";
import { authErrorLabels } from "@/features/auth/labels";
import type { AppLocale } from "@/lib/i18n/routing";
import { siteUrl } from "@/lib/site";

type ResetPasswordPageProps = Readonly<{
  params: Promise<{ locale: string }>;
}>;

export async function generateMetadata({
  params,
}: ResetPasswordPageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({
    locale: locale as AppLocale,
    namespace: "Auth.ResetPassword",
  });

  return {
    metadataBase: siteUrl,
    title: t("metaTitle"),
    description: t("metaDescription"),
  };
}

export default async function ResetPasswordPage({
  params,
}: ResetPasswordPageProps) {
  const { locale } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const [t, tErrors] = await Promise.all([
    getTranslations({ locale: appLocale, namespace: "Auth.ResetPassword" }),
    getTranslations({ locale: appLocale, namespace: "Auth.Errors" }),
  ]);

  return (
    <ResetPasswordForm
      locale={appLocale}
      labels={{
        titleLine1: t("titleLine1"),
        titleLine2: t("titleLine2"),
        titleAccent: t("titleAccent"),
        description: t("description"),
        email: t("email"),
        emailPlaceholder: t("emailPlaceholder"),
        otp: t("otp"),
        otpPlaceholder: t("otpPlaceholder"),
        newPassword: t("newPassword"),
        newPasswordPlaceholder: t("newPasswordPlaceholder"),
        confirmPassword: t("confirmPassword"),
        confirmPasswordPlaceholder: t("confirmPasswordPlaceholder"),
        submit: t("submit"),
        submitting: t("submitting"),
        rememberPassword: t("rememberPassword"),
        logIn: t("logIn"),
        successTitle: t("successTitle"),
        successDescription: t("successDescription"),
        backToLogin: t("backToLogin"),
        showPassword: t("showPassword"),
        hidePassword: t("hidePassword"),
        errors: authErrorLabels(tErrors),
      }}
    />
  );
}

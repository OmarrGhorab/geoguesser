import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { SignUpForm } from "@/features/auth/components/sign-up-form";
import { authErrorLabels } from "@/features/auth/labels";
import { getOAuthUrls } from "@/features/auth/oauth";
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
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const [t, tErrors] = await Promise.all([
    getTranslations({ locale: appLocale, namespace: "Auth.SignUp" }),
    getTranslations({ locale: appLocale, namespace: "Auth.Errors" }),
  ]);
  const oauth = getOAuthUrls();

  return (
    <SignUpForm
      locale={appLocale}
      googleOAuthUrl={oauth.google}
      discordOAuthUrl={oauth.discord}
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
        creatingAccount: t("creatingAccount"),
        alreadyHaveAccount: t("alreadyHaveAccount"),
        logIn: t("logIn"),
        showPassword: t("showPassword"),
        hidePassword: t("hidePassword"),
        errors: authErrorLabels(tErrors),
      }}
    />
  );
}

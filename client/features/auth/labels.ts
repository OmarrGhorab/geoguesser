import type { getTranslations } from "next-intl/server";

type Translator = Awaited<ReturnType<typeof getTranslations>>;

export function authErrorLabels(t: Translator): Record<string, string> {
  return {
    generic: t("generic"),
    emailRequired: t("emailRequired"),
    emailInvalid: t("emailInvalid"),
    passwordRequired: t("passwordRequired"),
    passwordTooShort: t("passwordTooShort"),
    passwordTooLong: t("passwordTooLong"),
    passwordMismatch: t("passwordMismatch"),
    confirmPasswordRequired: t("confirmPasswordRequired"),
    displayNameLength: t("displayNameLength"),
    otpInvalid: t("otpInvalid"),
    emailAlreadyExists: t("emailAlreadyExists"),
    invalidCredentials: t("invalidCredentials"),
    invalidOtp: t("invalidOtp"),
    invalidPassword: t("invalidPassword"),
    validationFailed: t("validationFailed"),
    rateLimited: t("rateLimited"),
  };
}

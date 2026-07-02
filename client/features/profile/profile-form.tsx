"use client";

import { useActionState } from "react";
import { useTranslations } from "next-intl";
import { updateProfileAction, type ProfileFormState } from "./actions";
import type { ProfileDTO } from "./types";

const INITIAL_STATE: ProfileFormState = { status: "idle" };

type ProfileFormProps = {
  profile: ProfileDTO;
};

export function ProfileForm({ profile }: ProfileFormProps) {
  const t = useTranslations("Profile");
  const [state, formAction, isPending] = useActionState(updateProfileAction, INITIAL_STATE);

  const fieldErrors = state.status === "error" ? state.fieldErrors ?? {} : {};
  const current = state.status === "success" ? state.profile : profile;

  return (
    <form action={formAction} className="grid gap-4" aria-describedby="profile-form-status">
      <label className="grid gap-1 text-sm">
        <span className="text-slate-600">{t("fields.displayName")}</span>
        <input
          className="rounded-md border border-slate-300 px-3 py-2 text-sm"
          name="display_name"
          type="text"
          minLength={2}
          maxLength={32}
          defaultValue={current.display_name}
          aria-invalid={Boolean(fieldErrors.display_name)}
          aria-describedby={fieldErrors.display_name ? "display_name-error" : undefined}
          required
        />
        {fieldErrors.display_name ? (
          <span id="display_name-error" className="text-sm text-red-700">
            {t(`validation.${fieldErrors.display_name}`)}
          </span>
        ) : null}
      </label>

      <label className="grid gap-1 text-sm">
        <span className="text-slate-600">{t("fields.avatarUrl")}</span>
        <input
          className="rounded-md border border-slate-300 px-3 py-2 text-sm"
          name="avatar_url"
          type="url"
          defaultValue={current.avatar_url ?? ""}
          aria-invalid={Boolean(fieldErrors.avatar_url)}
          aria-describedby={fieldErrors.avatar_url ? "avatar_url-error" : undefined}
        />
        {fieldErrors.avatar_url ? (
          <span id="avatar_url-error" className="text-sm text-red-700">
            {t(`validation.${fieldErrors.avatar_url}`)}
          </span>
        ) : null}
      </label>

      <label className="grid gap-1 text-sm">
        <span className="text-slate-600">{t("fields.countryCode")}</span>
        <input
          className="rounded-md border border-slate-300 px-3 py-2 text-sm uppercase"
          name="country_code"
          type="text"
          maxLength={2}
          defaultValue={current.country_code ?? ""}
          aria-invalid={Boolean(fieldErrors.country_code)}
          aria-describedby={fieldErrors.country_code ? "country_code-error" : undefined}
        />
        {fieldErrors.country_code ? (
          <span id="country_code-error" className="text-sm text-red-700">
            {t(`validation.${fieldErrors.country_code}`)}
          </span>
        ) : null}
      </label>

      <label className="grid gap-1 text-sm">
        <span className="text-slate-600">{t("fields.locale")}</span>
        <select
          className="rounded-md border border-slate-300 px-3 py-2 text-sm"
          name="locale"
          defaultValue={current.locale}
          aria-invalid={Boolean(fieldErrors.locale)}
          aria-describedby={fieldErrors.locale ? "locale-error" : undefined}
        >
          <option value="en">{t("localeOptions.en")}</option>
          <option value="ar">{t("localeOptions.ar")}</option>
        </select>
        {fieldErrors.locale ? (
          <span id="locale-error" className="text-sm text-red-700">
            {t(`validation.${fieldErrors.locale}`)}
          </span>
        ) : null}
      </label>

      <label className="grid gap-1 text-sm">
        <span className="text-slate-600">{t("fields.timezone")}</span>
        <input
          className="rounded-md border border-slate-300 px-3 py-2 text-sm"
          name="timezone"
          type="text"
          defaultValue={current.timezone ?? ""}
          aria-invalid={Boolean(fieldErrors.timezone)}
          aria-describedby={fieldErrors.timezone ? "timezone-error" : undefined}
        />
        {fieldErrors.timezone ? (
          <span id="timezone-error" className="text-sm text-red-700">
            {t(`validation.${fieldErrors.timezone}`)}
          </span>
        ) : null}
      </label>

      <button
        className="inline-flex w-fit items-center justify-center rounded-md bg-slate-950 px-4 py-2 text-sm font-semibold text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600 disabled:cursor-not-allowed disabled:opacity-50"
        type="submit"
        disabled={isPending}
      >
        {isPending ? t("actions.saving") : t("actions.save")}
      </button>

      <p id="profile-form-status" role="status" aria-live="polite" className="text-sm">
        {state.status === "success" ? t("actions.saved") : null}
        {state.status === "error" && state.code === "rate_limited" ? t("actions.rateLimited") : null}
        {state.status === "error" && state.code === "unauthorized" ? t("actions.unauthorized") : null}
        {state.status === "error" && state.code === "unexpected" ? t("actions.unexpected") : null}
        {state.status === "error" && state.code === "validation" ? t("actions.validationFailed") : null}
      </p>
    </form>
  );
}

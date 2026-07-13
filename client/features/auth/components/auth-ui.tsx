"use client";

import { useState } from "react";
import Link from "next/link";
import { useLocale } from "next-intl";
import { Eye, EyeOff } from "lucide-react";
import { motion, type Variants } from "motion/react";
import { AUTH_CONCEPT } from "@/features/auth/auth-concept";
import { DiscordIcon, GoogleIcon } from "@/features/auth/components/auth-icons";
import { oauthStartPath } from "@/features/auth/oauth";
import type { AppLocale } from "@/lib/i18n/routing";
import { cn } from "@/lib/utils";

export { AUTH_CONCEPT };

export const authFieldClassName = AUTH_CONCEPT.fieldClass;
export const authPasswordFieldClassName = AUTH_CONCEPT.passwordFieldClass;
export const authSocialButtonClassName = AUTH_CONCEPT.socialButtonClass;
export const authPrimaryButtonClassName = AUTH_CONCEPT.primaryButtonClass;
export const authLinkClassName = AUTH_CONCEPT.linkClass;
export const authMutedLinkClassName = AUTH_CONCEPT.mutedLinkClass;

export const containerVariants: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
      delayChildren: 0.08,
    },
  },
};

export const itemVariants: Variants = {
  hidden: { opacity: 0, y: 15 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      type: "spring",
      stiffness: 300,
      damping: 24,
    },
  },
};

type AuthPanelProps = {
  children: React.ReactNode;
  className?: string;
};

export function AuthBrandMark({ className }: { className?: string }) {
  const locale = useLocale();

  return (
    <AuthItem className={cn("mb-10 flex justify-start", className)}>
      <Link
        href={`/${locale}`}
        className="auth-brand-logo focus-visible:ring-auth-accent/70 focus-visible:ring-offset-auth-bg rounded-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
        aria-label={`Go to ${AUTH_CONCEPT.logoAlt} home`}
      />
    </AuthItem>
  );
}

export function AuthPanel({ children, className }: AuthPanelProps) {
  return (
    <motion.div
      key="auth-panel"
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      className={cn("relative z-10 w-full max-w-[490px]", className)}
    >
      <AuthBrandMark />
      {children}
    </motion.div>
  );
}

export function AuthItem({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <motion.div variants={itemVariants} className={className}>
      {children}
    </motion.div>
  );
}

type AuthTitleProps = {
  line1: string;
  line2?: string;
  accent?: string;
  description?: string;
};

export function AuthTitle({
  line1,
  line2,
  accent,
  description,
}: AuthTitleProps) {
  const hasSecondLine = Boolean(line2?.trim() || accent?.trim());

  return (
    <AuthItem className="mb-7 text-start">
      <h1 className="font-auth-display text-[2.6rem] leading-[1.06] font-black tracking-[0.01em] text-balance text-white uppercase italic sm:text-[3rem]">
        {line1}
        {hasSecondLine ? (
          <>
            <br />
            {line2?.trim() ? (
              <>
                {line2}
                {accent?.trim() ? " " : null}
              </>
            ) : null}
            {accent?.trim() ? (
              <span className={AUTH_CONCEPT.accentTextClass}>{accent}</span>
            ) : null}
          </>
        ) : null}
      </h1>
      {description ? (
        <p className="mt-3 text-sm leading-relaxed text-balance text-neutral-400 normal-case">
          {description}
        </p>
      ) : null}
    </AuthItem>
  );
}

type AuthSocialButtonsProps = {
  googleLabel: string;
  discordLabel: string;
  locale: AppLocale;
};

export function AuthSocialButtons({
  googleLabel,
  discordLabel,
  locale,
}: AuthSocialButtonsProps) {
  return (
    <AuthItem className="mb-7 grid grid-cols-1 gap-4 sm:grid-cols-2 sm:gap-4">
      <a
        href={oauthStartPath("google", locale)}
        className={cn(
          authSocialButtonClassName,
          "border-white bg-white text-[#080b20] shadow-[0_8px_24px_rgba(0,0,0,0.18)] hover:brightness-95",
        )}
      >
        <GoogleIcon className="text-[16px]" />
        {googleLabel}
      </a>
      <a
        href={oauthStartPath("discord", locale)}
        className={cn(
          authSocialButtonClassName,
          "border-[#7280ff] bg-[#5865F2] text-white shadow-[0_8px_24px_rgba(88,101,242,0.3)] hover:brightness-110",
        )}
      >
        <DiscordIcon className="text-[16px] text-white" />
        {discordLabel}
      </a>
    </AuthItem>
  );
}

export function AuthDivider({ label }: { label: string }) {
  return (
    <AuthItem className="relative mb-5 flex items-center">
      <div className="grow border-t border-white/10" />
      <span className="px-4 text-xs font-medium text-neutral-300 uppercase">
        {label}
      </span>
      <div className="grow border-t border-white/10" />
    </AuthItem>
  );
}

type AuthFieldProps = {
  id: string;
  name: string;
  label: string;
  type?: React.HTMLInputTypeAttribute;
  placeholder?: string;
  autoComplete?: string;
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  inputMode?: React.HTMLAttributes<HTMLInputElement>["inputMode"];
  spellCheck?: boolean;
  defaultValue?: string;
  error?: string;
  disabled?: boolean;
};

export function AuthField({
  id,
  name,
  label,
  type = "text",
  placeholder,
  autoComplete,
  required,
  minLength,
  maxLength,
  pattern,
  inputMode,
  spellCheck,
  defaultValue,
  error,
  disabled,
}: AuthFieldProps) {
  if (type === "password") {
    return (
      <AuthPasswordField
        id={id}
        name={name}
        label={label}
        placeholder={placeholder}
        autoComplete={autoComplete}
        required={required}
        minLength={minLength}
        maxLength={maxLength}
        defaultValue={defaultValue}
        error={error}
        disabled={disabled}
      />
    );
  }

  const errorId = error ? `${id}-error` : undefined;

  return (
    <AuthItem className="flex flex-col gap-2">
      <label htmlFor={id} className="text-sm font-medium text-neutral-200">
        {label}
      </label>
      <input
        id={id}
        name={name}
        type={type}
        placeholder={placeholder}
        autoComplete={autoComplete}
        required={required}
        minLength={minLength}
        maxLength={maxLength}
        pattern={pattern}
        inputMode={inputMode}
        spellCheck={spellCheck}
        defaultValue={defaultValue}
        disabled={disabled}
        aria-invalid={error ? true : undefined}
        aria-describedby={errorId}
        className={authFieldClassName}
      />
      {error ? (
        <p id={errorId} role="alert" className="text-destructive text-xs">
          {error}
        </p>
      ) : null}
    </AuthItem>
  );
}

type AuthPasswordFieldProps = {
  id: string;
  name: string;
  label: string;
  placeholder?: string;
  autoComplete?: string;
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  defaultValue?: string;
  labelEnd?: React.ReactNode;
  showPasswordLabel?: string;
  hidePasswordLabel?: string;
  error?: string;
  disabled?: boolean;
};

export function AuthPasswordField({
  id,
  name,
  label,
  placeholder,
  autoComplete,
  required,
  minLength,
  maxLength,
  defaultValue,
  labelEnd,
  showPasswordLabel = "Show password",
  hidePasswordLabel = "Hide password",
  error,
  disabled,
}: AuthPasswordFieldProps) {
  const [visible, setVisible] = useState(false);
  const errorId = error ? `${id}-error` : undefined;

  return (
    <AuthItem className="flex flex-col gap-2">
      <div className="flex items-center justify-between gap-3">
        <label htmlFor={id} className="text-sm font-medium text-neutral-200">
          {label}
        </label>
        {labelEnd}
      </div>
      <div className="relative">
        <input
          id={id}
          name={name}
          type={visible ? "text" : "password"}
          placeholder={placeholder}
          autoComplete={autoComplete}
          required={required}
          minLength={minLength}
          maxLength={maxLength}
          defaultValue={defaultValue}
          disabled={disabled}
          aria-invalid={error ? true : undefined}
          aria-describedby={errorId}
          className={authPasswordFieldClassName}
        />
        <button
          type="button"
          onClick={() => setVisible((current) => !current)}
          disabled={disabled}
          className="text-muted-foreground hover:text-foreground focus-visible:ring-auth-accent/50 absolute end-3 top-1/2 -translate-y-1/2 rounded-md p-1 transition-colors focus-visible:ring-2 focus-visible:outline-none disabled:opacity-50"
          aria-label={visible ? hidePasswordLabel : showPasswordLabel}
          aria-pressed={visible}
        >
          {visible ? (
            <EyeOff className="size-4" aria-hidden="true" />
          ) : (
            <Eye className="size-4" aria-hidden="true" />
          )}
        </button>
      </div>
      {error ? (
        <p id={errorId} role="alert" className="text-destructive text-xs">
          {error}
        </p>
      ) : null}
    </AuthItem>
  );
}

export function AuthFormError({ message }: { message?: string }) {
  if (!message) return null;
  return (
    <AuthItem>
      <p
        role="alert"
        className="border-destructive/40 bg-destructive/10 text-destructive rounded-2xl border px-4 py-3 text-sm"
      >
        {message}
      </p>
    </AuthItem>
  );
}

export function AuthPrimaryButton({
  children,
  type = "submit",
  disabled,
  pendingLabel,
  isPending,
}: {
  children: React.ReactNode;
  type?: "button" | "submit" | "reset";
  disabled?: boolean;
  pendingLabel?: string;
  isPending?: boolean;
}) {
  return (
    <AuthItem className="mt-4">
      <button
        type={type}
        disabled={disabled || isPending}
        aria-busy={isPending || undefined}
        className={authPrimaryButtonClassName}
      >
        {isPending && pendingLabel ? pendingLabel : children}
      </button>
    </AuthItem>
  );
}

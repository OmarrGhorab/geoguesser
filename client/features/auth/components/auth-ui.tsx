"use client";

import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { motion, type Variants } from "motion/react";
import { DiscordIcon, GoogleIcon } from "@/features/auth/components/auth-icons";
import { Link } from "@/lib/i18n/link";
import { cn } from "@/lib/utils";

export const authFieldClassName =
  "w-full rounded-[14px] border border-border bg-surface px-4 py-3.5 text-sm text-foreground transition-colors placeholder:text-muted-foreground focus:border-ring focus:bg-card focus:ring-1 focus:ring-ring focus:outline-none aria-invalid:border-destructive aria-invalid:ring-1 aria-invalid:ring-destructive/40";

export const authPasswordFieldClassName =
  "w-full rounded-[14px] border border-border bg-surface py-3.5 ps-4 pe-12 text-sm text-foreground transition-colors placeholder:text-muted-foreground focus:border-ring focus:bg-card focus:ring-1 focus:ring-ring focus:outline-none aria-invalid:border-destructive aria-invalid:ring-1 aria-invalid:ring-destructive/40";

export const authSocialButtonClassName =
  "flex items-center justify-center gap-2 rounded-full border border-border bg-card py-3 text-[13px] font-medium text-foreground transition-colors hover:bg-muted active:scale-[0.96]";

export const authPrimaryButtonClassName =
  "w-full rounded-full bg-primary py-3.5 text-sm font-medium text-primary-foreground shadow-[0_0_20px_rgba(255,255,255,0.05)] transition-colors hover:bg-primary/90 active:scale-[0.96] disabled:pointer-events-none disabled:opacity-60";

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

export function AuthPanel({ children, className }: AuthPanelProps) {
  return (
    <motion.div
      key="auth-panel"
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      className={cn("w-full max-w-[400px]", className)}
    >
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
    <AuthItem className="mb-10 text-center">
      <h1 className="text-3xl leading-tight font-medium tracking-tight text-balance text-white md:text-[40px]">
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
              <span className="font-serif font-light italic">{accent}</span>
            ) : null}
          </>
        ) : null}
      </h1>
      {description ? (
        <p className="mt-3 text-sm leading-relaxed text-balance text-neutral-400">
          {description}
        </p>
      ) : null}
    </AuthItem>
  );
}

type AuthSocialButtonsProps = {
  googleLabel: string;
  discordLabel: string;
  googleHref: string;
  discordHref: string;
};

export function AuthSocialButtons({
  googleLabel,
  discordLabel,
  googleHref,
  discordHref,
}: AuthSocialButtonsProps) {
  return (
    <AuthItem className="mb-8 grid grid-cols-2 gap-4">
      <a href={googleHref} className={authSocialButtonClassName}>
        <GoogleIcon className="text-[16px]" />
        {googleLabel}
      </a>
      <a href={discordHref} className={authSocialButtonClassName}>
        <DiscordIcon className="text-[16px] text-[#5865F2]" />
        {discordLabel}
      </a>
    </AuthItem>
  );
}

export function AuthDivider({ label }: { label: string }) {
  return (
    <AuthItem className="relative mb-8 flex items-center">
      <div className="grow border-t border-white/10" />
      <span className="px-4 text-[11px] font-medium tracking-wider text-neutral-500 uppercase">
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
      <label htmlFor={id} className="text-foreground text-sm font-medium">
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
        <label htmlFor={id} className="text-foreground text-sm font-medium">
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
          className="text-muted-foreground hover:text-foreground focus-visible:ring-ring absolute end-3 top-1/2 -translate-y-1/2 rounded-md p-1 transition-colors focus-visible:ring-2 focus-visible:outline-none disabled:opacity-50"
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
        className="border-destructive/40 bg-destructive/10 text-destructive rounded-[14px] border px-4 py-3 text-sm"
      >
        {message}
      </p>
    </AuthItem>
  );
}

type AuthHref = React.ComponentProps<typeof Link>["href"];

type AuthFooterProps = {
  prompt: string;
  href: AuthHref;
  linkLabel: string;
};

export function AuthFooter({ prompt, href, linkLabel }: AuthFooterProps) {
  return (
    <AuthItem>
      <p className="mt-6 text-[13px] text-neutral-400">
        {prompt}{" "}
        <Link href={href} className="font-bold text-white hover:underline">
          {linkLabel}
        </Link>
      </p>
    </AuthItem>
  );
}

export type { AuthHref };

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

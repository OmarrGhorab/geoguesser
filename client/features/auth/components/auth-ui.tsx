"use client";

import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { motion, type Variants } from "motion/react";
import { DiscordIcon, GoogleIcon } from "@/features/auth/components/auth-icons";
import { cn } from "@/lib/utils";

export const authFieldClassName =
  "w-full rounded-[14px] border border-white/10 bg-[#0A0A0A] px-4 py-3.5 text-sm text-white transition-colors placeholder:text-neutral-500 focus:border-neutral-500 focus:bg-[#111] focus:ring-1 focus:ring-neutral-500 focus:outline-none";

export const authPasswordFieldClassName =
  "w-full rounded-[14px] border border-white/10 bg-[#0A0A0A] py-3.5 ps-4 pe-12 text-sm text-white transition-colors placeholder:text-neutral-500 focus:border-neutral-500 focus:bg-[#111] focus:ring-1 focus:ring-neutral-500 focus:outline-none";

export const authSocialButtonClassName =
  "flex items-center justify-center gap-2 rounded-full border border-white/10 bg-[#141414] py-3 text-[13px] font-medium text-white transition-colors hover:bg-[#1f1f1f] active:scale-[0.96]";

export const authPrimaryButtonClassName =
  "w-full rounded-full bg-[#EAEAEA] py-3.5 text-sm font-medium text-black shadow-[0_0_20px_rgba(255,255,255,0.05)] transition-colors hover:bg-white active:scale-[0.96] disabled:pointer-events-none disabled:opacity-60";

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

export function AuthTitle({ line1, line2, accent, description }: AuthTitleProps) {
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
};

export function AuthSocialButtons({
  googleLabel,
  discordLabel,
}: AuthSocialButtonsProps) {
  return (
    <AuthItem className="mb-8 grid grid-cols-2 gap-4">
      <button type="button" className={authSocialButtonClassName}>
        <GoogleIcon className="text-[16px]" />
        {googleLabel}
      </button>
      <button type="button" className={authSocialButtonClassName}>
        <DiscordIcon className="text-[16px] text-[#5865F2]" />
        {discordLabel}
      </button>
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
      />
    );
  }

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
        className={authFieldClassName}
      />
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
}: AuthPasswordFieldProps) {
  const [visible, setVisible] = useState(false);

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
          className={authPasswordFieldClassName}
        />
        <button
          type="button"
          onClick={() => setVisible((current) => !current)}
          className="absolute end-3 top-1/2 -translate-y-1/2 rounded-md p-1 text-neutral-400 transition-colors hover:text-white focus-visible:ring-2 focus-visible:ring-neutral-500 focus-visible:outline-none"
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
    </AuthItem>
  );
}

type AuthFooterProps = {
  prompt: string;
  href: string;
  linkLabel: string;
};

export function AuthFooter({ prompt, href, linkLabel }: AuthFooterProps) {
  return (
    <AuthItem>
      <p className="mt-6 text-[13px] text-neutral-400">
        {prompt}{" "}
        <a href={href} className="font-bold text-white hover:underline">
          {linkLabel}
        </a>
      </p>
    </AuthItem>
  );
}

export function AuthPrimaryButton({
  children,
  type = "submit",
}: {
  children: React.ReactNode;
  type?: "button" | "submit" | "reset";
}) {
  return (
    <AuthItem className="mt-4">
      <button type={type} className={authPrimaryButtonClassName}>
        {children}
      </button>
    </AuthItem>
  );
}

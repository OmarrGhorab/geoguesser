"use client";

import { useSyncExternalStore } from "react";

type GlobalErrorProps = Readonly<{
  error: Error & { digest?: string };
  reset: () => void;
}>;

/**
 * Required root error UI for App Router. Without this, some Next 16 + Turbopack
 * dev builds fail resolving the built-in global-error client module when the
 * browser origin is not localhost (e.g. ngrok).
 */
export default function GlobalError({ error, reset }: GlobalErrorProps) {
  const locale = useSyncExternalStore(
    () => () => undefined,
    () => (window.location.pathname.startsWith("/ar") ? "ar" : "en"),
    () => "en",
  );

  const copy =
    locale === "ar"
      ? {
          title: "حدث خطأ ما",
          description: "حاول تحديث الصفحة. إذا استمرت المشكلة، تواصل مع الدعم.",
          errorId: "معرّف الخطأ",
          retry: "حاول مرة أخرى",
        }
      : {
          title: "Something went wrong",
          description:
            "Try refreshing the page. If the problem continues, contact support.",
          errorId: "Error ID",
          retry: "Try again",
        };

  return (
    <html lang={locale} dir={locale === "ar" ? "rtl" : "ltr"}>
      <head>
        <title>{copy.title}</title>
      </head>
      <body
        style={{
          margin: 0,
          minHeight: "100dvh",
          display: "grid",
          placeItems: "center",
          fontFamily: "system-ui, sans-serif",
          background: "#0a0a0a",
          color: "#fafafa",
        }}
      >
        <main style={{ textAlign: "center", padding: "2rem" }}>
          <h1 style={{ fontSize: "1.25rem", fontWeight: 600 }}>{copy.title}</h1>
          <p style={{ opacity: 0.7, marginTop: "0.75rem" }}>
            {error.digest
              ? `${copy.errorId}: ${error.digest}`
              : copy.description}
          </p>
          <button
            type="button"
            onClick={reset}
            style={{
              marginTop: "1.5rem",
              border: "1px solid rgba(255,255,255,0.2)",
              borderRadius: "999px",
              background: "transparent",
              color: "inherit",
              padding: "0.6rem 1.25rem",
              cursor: "pointer",
            }}
          >
            {copy.retry}
          </button>
        </main>
      </body>
    </html>
  );
}

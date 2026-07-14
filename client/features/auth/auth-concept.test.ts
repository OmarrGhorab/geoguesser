import { describe, expect, it } from "vitest";
import { AUTH_CONCEPT } from "@/features/auth/auth-concept";

/**
 * Presentational contract for the auth form concept (right panel).
 * Asserts shipped class tokens match the blue journey concept, not legacy white primary.
 * Tests the real AUTH_CONCEPT module consumed by auth-ui.
 */
describe("AUTH_CONCEPT presentational tokens", () => {
  it("uses the current WorldGuesser logo on the form side", () => {
    expect(AUTH_CONCEPT.logoSrc).toBe("/logo-3.png");
    expect(AUTH_CONCEPT.logoAlt).toBe("WorldGuesser");
  });

  it("primary CTA uses the Discord blue pill, not near-white bg-primary", () => {
    expect(AUTH_CONCEPT.primaryButtonClass).toContain("#5865F2");
    expect(AUTH_CONCEPT.primaryButtonClass).toContain("rounded-full");
    expect(AUTH_CONCEPT.primaryButtonClass).not.toContain("bg-primary");
  });

  it("title accent and secondary links use concept accent colors", () => {
    expect(AUTH_CONCEPT.accentTextClass).toContain("text-auth-accent");
    expect(AUTH_CONCEPT.linkClass).toContain("text-auth-link");
    expect(AUTH_CONCEPT.linkClass).not.toContain("text-white");
  });

  it("fields and social buttons use the concept's rounded rectangle borders", () => {
    expect(AUTH_CONCEPT.fieldClass).toContain("rounded-xl");
    expect(AUTH_CONCEPT.fieldClass).toContain("border-auth-field-border");
    expect(AUTH_CONCEPT.passwordFieldClass).toContain("rounded-xl");
    expect(AUTH_CONCEPT.socialButtonClass).toContain("rounded-full");
  });
});

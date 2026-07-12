import { expect, test } from "@playwright/test";

test.describe("auth UI", () => {
  test("sign-up page renders backend-aligned fields and password toggles", async ({
    page,
  }) => {
    await page.goto("/en/sign-up");

    await expect(
      page.getByRole("heading", { name: /Explore the world/i }),
    ).toBeVisible();
    await expect(page.getByLabel("Display name")).toBeVisible();
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.locator("#sign-up-password")).toBeVisible();
    await expect(page.locator("#sign-up-confirm-password")).toBeVisible();

    await page.getByRole("button", { name: "Show password" }).first().click();
    await expect(page.locator("#sign-up-password")).toHaveAttribute(
      "type",
      "text",
    );
  });

  test("sign-up shows validation when passwords do not match", async ({
    page,
  }) => {
    await page.goto("/en/sign-up");

    await page.getByLabel("Display name").fill("Explorer");
    await page.getByLabel("Email").fill("player@example.com");
    await page.locator("#sign-up-password").fill("super-secure-1");
    await page.locator("#sign-up-confirm-password").fill("super-secure-2");
    await page.getByRole("button", { name: "Create account" }).click();

    await expect(page.locator("#sign-up-confirm-password-error")).toContainText(
      /do not match/i,
      { timeout: 15_000 },
    );
  });

  test("login page exposes forgot-password client navigation", async ({
    page,
  }) => {
    await page.goto("/en/login");
    await page.getByRole("link", { name: "Forgot password?" }).click();
    await expect(page).toHaveURL(/\/en\/forgot-password$/, { timeout: 15_000 });
    await expect(
      page.getByRole("heading", { name: /Forgot your/i }),
    ).toBeVisible();
  });

  test("reset password page does not put email in the URL", async ({
    page,
  }) => {
    await page.goto("/en/reset-password");
    await expect(page).toHaveURL(/\/en\/reset-password$/);
    await expect(page.url()).not.toContain("email=");
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.getByLabel("Reset code")).toBeVisible();
    await expect(page.locator("#reset-new-password")).toBeVisible();
  });

  test("oauth buttons point at backend oauth routes", async ({ page }) => {
    await page.goto("/en/login");
    await expect(
      page.getByRole("link", { name: "Continue with Google" }),
    ).toHaveAttribute("href", "/api/auth/oauth/google?locale=en");
    await expect(
      page.getByRole("link", { name: "Continue with Discord" }),
    ).toHaveAttribute("href", "/api/auth/oauth/discord?locale=en");
  });

  test("oauth callback failures return to localized login", async ({
    page,
  }) => {
    await page.goto("/en/login?oauth_error=1");
    await expect(
      page.getByRole("alert").filter({
        hasText: /social sign-in could not be completed/i,
      }),
    ).toContainText(/social sign-in could not be completed/i);
  });
});

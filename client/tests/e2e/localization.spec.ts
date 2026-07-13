import { expect, test } from "@playwright/test";

test("redirects the root route to the default locale", async ({ page }) => {
  await page.goto("/");

  await expect(page).toHaveURL(/\/en$/);
  await expect(
    page.getByRole("heading", { level: 2, name: "Explore the world" }),
  ).toBeVisible();
});

test("sets English document language and direction", async ({ page }) => {
  await page.goto("/en");

  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await expect(page.locator("html")).toHaveAttribute("dir", "ltr");
});

test("sets Arabic document language and direction", async ({ page }) => {
  await page.goto("/ar");

  await expect(page.locator("html")).toHaveAttribute("lang", "ar");
  await expect(page.locator("html")).toHaveAttribute("dir", "rtl");
  await expect(
    page.getByRole("heading", { level: 2, name: "استكشف العالم" }),
  ).toBeVisible();
});

test("returns not found for unsupported locales", async ({ page }) => {
  const response = await page.goto("/fr");

  expect(response?.status()).toBe(404);
});

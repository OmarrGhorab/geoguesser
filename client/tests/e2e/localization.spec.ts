import { expect, test } from "@playwright/test";

test("redirects the root route to the default locale", async ({ page }) => {
  await page.goto("/");

  await expect(page).toHaveURL(/\/en$/);
  await expect(
    page.getByRole("heading", { level: 1, name: "GeoGuess" }),
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
    page.getByRole("heading", { level: 1, name: "جيوجيس" }),
  ).toBeVisible();
});

test("returns not found for unsupported locales", async ({ page }) => {
  const response = await page.goto("/fr");

  expect(response?.status()).toBe(404);
});

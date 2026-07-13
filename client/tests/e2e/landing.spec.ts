import { expect, test } from "@playwright/test";

test.describe("public landing page", () => {
  test("signed-out visitors see all four localized landing sections", async ({
    page,
  }) => {
    await page.goto("/en");

    await expect(page.getByTestId("public-landing")).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Explore the world" }),
    ).toHaveCount(2);
    await expect(
      page.getByRole("heading", { name: "Play with Friends" }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Compete against others" }),
    ).toBeVisible();
  });

  test("landing navigation links target the matching sections", async ({
    page,
  }) => {
    await page.goto("/en");

    await expect(page.getByRole("link", { name: "Explore" })).toHaveAttribute(
      "href",
      "#explore",
    );
    await expect(
      page.getByRole("link", { name: "Multiplayer" }),
    ).toHaveAttribute("href", "#friends");
    await expect(
      page.getByRole("link", { name: "Leaderboards" }),
    ).toHaveAttribute("href", "#compete");
    await expect(page.getByRole("link", { name: "Pricing" })).toHaveAttribute(
      "href",
      "#pricing",
    );
    await expect(page.getByRole("link", { name: "Login" })).toHaveAttribute(
      "href",
      "/en/login",
    );
    await expect(page.getByRole("link", { name: "Play Free" })).toHaveAttribute(
      "href",
      "/en/sign-up",
    );
  });

  test("uses layered video, background, character, and text assets", async ({
    page,
  }) => {
    await page.goto("/en");

    await expect(page.locator("#explore video source")).toHaveAttribute(
      "src",
      "/landing/hero-vid.mp4",
    );
    await expect(page.locator("#explore img")).toHaveCount(0);
    await expect(
      page.locator("#discover img[src*='section2-bg']"),
    ).toBeVisible();
    await expect(
      page.locator("#friends img[src*='section3-bg']"),
    ).toBeVisible();
    await expect(
      page.getByRole("img", {
        name: "Two friends fist-bumping",
      }),
    ).toHaveAttribute("src", /friends-city\.png/);
    await expect(page.locator("#friends")).toContainText(
      "Put your skills to the test against your friends and family.",
    );
    await expect(
      page.locator("#compete img[src*='section4-bg']"),
    ).toBeVisible();
    await expect(
      page.getByRole("img", {
        name: "Two competitors facing off in a friendly challenge",
      }),
    ).toHaveAttribute("src", /compete-arena\.png/);
    await expect(
      page.getByRole("img", { name: "A traveler holding a glowing map" }),
    ).toHaveAttribute("src", /character-secondsection\.png/);
    await expect(page.locator("#discover")).toContainText(
      "Get dropped anywhere from the busy streets of New York to the beautiful beaches of Bali.",
    );
  });

  test("sections reveal as visitors scroll to them", async ({ page }) => {
    await page.goto("/en");

    const competeSection = page.locator("#compete");
    await expect(competeSection).toHaveCSS("opacity", "0");

    await competeSection.scrollIntoViewIfNeeded();
    await expect(competeSection).toHaveCSS("opacity", "1");
  });

  test("authenticated sessions keep the signed-in home", async ({
    context,
    page,
  }) => {
    await context.addCookies([
      {
        name: "access_token",
        value: "test-session",
        domain: "127.0.0.1",
        path: "/",
        httpOnly: true,
        sameSite: "Lax",
      },
    ]);

    await page.goto("/en");

    await expect(page.getByTestId("public-landing")).toHaveCount(0);
    await expect(page.getByRole("heading", { name: "GeoGuess" })).toBeVisible();
  });
});

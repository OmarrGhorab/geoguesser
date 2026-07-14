import { expect, test } from "@playwright/test";

test.describe("public landing page", () => {
  test("signed-out visitors see all four localized landing sections", async ({
    page,
  }) => {
    await page.goto("/en");

    await expect(page.getByTestId("public-landing")).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Explore the world" }),
    ).toHaveCount(1);
    await expect(
      page.getByRole("heading", { name: "Discover new places" }),
    ).toBeVisible();
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

    const landingNav = page.getByRole("navigation", { name: "Landing page" });
    await expect(
      landingNav.getByRole("link", { name: "Explore" }),
    ).toHaveAttribute("href", "#explore");
    await expect(
      landingNav.getByRole("link", { name: "Multiplayer" }),
    ).toHaveAttribute("href", "#friends");
    await expect(
      landingNav.getByRole("link", { name: "Leaderboards" }),
    ).toHaveAttribute("href", "#compete");
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
    await expect(
      page.getByRole("img", { name: "WorldGuess", exact: true }),
    ).toHaveAttribute("src", /logo-2/);
    await expect(
      page.locator("#discover img[src*='section2-bg']"),
    ).toBeVisible();
    await expect(page.locator("#friends img[src*='section3-bg']")).toHaveCount(
      0,
    );
    await expect(
      page.locator("#friends img[alt='Two friends fist-bumping']"),
    ).toHaveAttribute("src", /friends-city\.png/);
    await expect(page.locator("#friends")).toContainText(
      "Put your skills to the test against your friends and family.",
    );
    const competeBackground = page.locator("#compete img[src*='section4-bg']");
    await expect(competeBackground).toBeVisible();
    await expect(competeBackground.locator("..")).toHaveCSS("opacity", "0.42");
    const competeArena = page.locator(
      "#compete img[alt='Two competitors facing off in a friendly challenge']",
    );
    await expect(competeArena).toHaveAttribute("src", /compete-arena\.png/);
    await expect(competeArena).toHaveCSS("object-fit", "cover");
    await expect(page.locator("#compete-title")).toHaveCSS(
      "text-align",
      "center",
    );
    const [arenaBox, titleBox] = await Promise.all([
      competeArena.boundingBox(),
      page.locator("#compete-title").boundingBox(),
    ]);
    expect(arenaBox).not.toBeNull();
    expect(titleBox).not.toBeNull();
    if (!arenaBox || !titleBox) {
      throw new Error("Competition artwork and title must have layout boxes");
    }
    expect(titleBox.y).toBeGreaterThanOrEqual(arenaBox.y + arenaBox.height);
    expect(titleBox.y - (arenaBox.y + arenaBox.height)).toBeLessThanOrEqual(12);
    await expect(
      page.locator("#discover img[alt='A traveler holding a glowing map']"),
    ).toHaveAttribute("src", /character-secondsection\.png/);
    await expect(page.locator("#discover")).toContainText(
      "Get dropped anywhere from the busy streets of New York to the beautiful beaches of Bali.",
    );
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
    await expect(
      page.getByRole("heading", { name: "WorldGuess" }),
    ).toBeVisible();
  });
});

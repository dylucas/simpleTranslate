import playwright from "@playwright/test";

const { expect, test } = playwright;

test("main translation workflow remains usable", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByLabel("应用侧边栏")).toBeVisible();
  await expect(page.getByText("预览")).toBeVisible();

  const dimensions = await page.evaluate(() => ({
    bodyWidth: document.body.scrollWidth,
    viewportWidth: document.documentElement.clientWidth,
  }));
  expect(dimensions.bodyWidth).toBeLessThanOrEqual(dimensions.viewportWidth);

  await page.getByLabel("打开设置").click();
  await page.getByPlaceholder("TokenHub API Key (sk-...)").fill("sk-preview");
  await page.getByRole("button", { name: "保存配置" }).click();
  await expect(page.getByRole("dialog", { name: "偏好设置" })).toBeHidden();

  await page.getByRole("textbox", { name: "原文" }).fill("hello");
  await page.getByRole("button", { name: "翻译", exact: true }).click();
  await expect(page.getByRole("textbox", { name: "译文" })).toHaveValue("你好");
  await expect(page.locator(".status")).toContainText("完成");

  await page.getByLabel("打开历史记录").click();
  await expect(page.getByRole("dialog", { name: "历史记录" })).toBeVisible();
  await expect(page.getByText("hello", { exact: true })).toBeVisible();
});

test("theme and settings dialogs remain keyboard accessible", async ({ page }) => {
  await page.goto("/");
  await page.getByLabel("切换深浅主题").click();
  await expect(page.locator(".app-shell")).toHaveClass(/light-mode/);

  const settingsTrigger = page.getByLabel("打开设置");
  await settingsTrigger.click();
  await expect(page.getByRole("dialog", { name: "偏好设置" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "偏好设置" })).toBeHidden();
  await expect(settingsTrigger).toBeFocused();
});

test("translation modes remain independent controls", async ({ page }) => {
  await page.goto("/");

  const auto = page.getByRole("button", { name: "自动翻译" });
  const clipboard = page.getByRole("button", { name: "剪贴板监听" });
  const compare = page.getByRole("button", { name: "多引擎对照" });

  await expect(auto).toHaveAttribute("aria-pressed", "true");
  await expect(clipboard).toHaveAttribute("aria-pressed", "false");
  await expect(compare).toHaveAttribute("aria-pressed", "false");

  await auto.click();
  await clipboard.click();
  await compare.click();

  await expect(auto).toHaveAttribute("aria-pressed", "false");
  await expect(clipboard).toHaveAttribute("aria-pressed", "true");
  await expect(compare).toHaveAttribute("aria-pressed", "true");
});

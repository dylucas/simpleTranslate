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

  const historyTrigger = page.getByLabel("打开历史记录");
  await historyTrigger.click();
  const historyDialog = page.getByRole("dialog", { name: "历史记录" });
  const historySearch = page.getByRole("textbox", { name: "搜索翻译记录" });
  await expect(historyDialog).toBeVisible();
  await expect(historySearch).toBeFocused();
  await expect(historyDialog.getByText("自动识别", { exact: true })).toBeVisible();
  await expect(historyDialog.getByText("中文", { exact: true })).toBeVisible();
  await expect(page.getByText("hello", { exact: true })).toBeVisible();

  if (await page.evaluate(() => matchMedia("(min-width: 1180px)").matches)) {
    await expect.poll(() => page.evaluate(() => {
      const drawer = document.querySelector<HTMLElement>(".drawer");
      const main = document.querySelector<HTMLElement>(".main-content");
      return Math.abs((main?.getBoundingClientRect().right ?? 0) - (drawer?.getBoundingClientRect().left ?? 0));
    })).toBeLessThanOrEqual(1);
  }

  await historySearch.fill("hello");
  await expect(historyDialog.getByText("1 条匹配记录", { exact: true })).toBeVisible();

  const historyLayout = await page.evaluate(() => {
    const drawer = document.querySelector<HTMLElement>(".drawer");
    const backdrop = document.querySelector<HTMLElement>(".backdrop");
    const main = document.querySelector<HTMLElement>(".main-content");
    const header = drawer?.querySelector<HTMLElement>("header");
    const card = drawer?.querySelector<HTMLElement>(".history-item");
    return {
      viewportWidth: document.documentElement.clientWidth,
      bodyWidth: document.body.scrollWidth,
      drawerWidth: drawer?.getBoundingClientRect().width ?? 0,
      drawerLeft: drawer?.getBoundingClientRect().left ?? 0,
      mainRight: main?.getBoundingClientRect().right ?? 0,
      backdropRight: backdrop?.getBoundingClientRect().right ?? 0,
      backdropBackground: backdrop ? getComputedStyle(backdrop).backgroundColor : "",
      docked: matchMedia("(min-width: 1180px)").matches,
      headerClientWidth: header?.clientWidth ?? 0,
      headerScrollWidth: header?.scrollWidth ?? 0,
      sourceClamp: card ? getComputedStyle(card.querySelector("strong")!).webkitLineClamp : "",
      outputClamp: card ? getComputedStyle(card.querySelector(".output")!).webkitLineClamp : "",
    };
  });
  expect(historyLayout.bodyWidth).toBeLessThanOrEqual(historyLayout.viewportWidth);
  expect(historyLayout.drawerWidth).toBeLessThanOrEqual(Math.min(440, historyLayout.viewportWidth));
  expect(historyLayout.headerScrollWidth).toBeLessThanOrEqual(historyLayout.headerClientWidth);
  expect(historyLayout.sourceClamp).toBe("2");
  expect(historyLayout.outputClamp).toBe("2");
  if (historyLayout.docked) {
    expect(Math.abs(historyLayout.mainRight - historyLayout.drawerLeft)).toBeLessThanOrEqual(1);
    expect(Math.abs(historyLayout.backdropRight - historyLayout.drawerLeft)).toBeLessThanOrEqual(1);
    expect(historyLayout.backdropBackground).toBe("rgba(0, 0, 0, 0)");
  } else {
    expect(historyLayout.mainRight).toBeGreaterThan(historyLayout.drawerLeft);
    expect(historyLayout.backdropRight).toBe(historyLayout.viewportWidth);
  }

  await page.getByText("hello", { exact: true }).click();
  await expect(historyDialog).toBeHidden();
  await expect(page.getByRole("textbox", { name: "原文" })).toHaveValue("hello");
  await expect(historyTrigger).toBeFocused();
});

test("theme and settings dialogs remain keyboard accessible", async ({ page }) => {
  await page.goto("/");
  await page.getByLabel("切换深浅主题").click();
  await expect(page.locator(".app-shell")).toHaveClass(/light-mode/);

  const historyTrigger = page.getByLabel("打开历史记录");
  await historyTrigger.click();
  await expect(page.getByRole("dialog", { name: "历史记录" })).toBeVisible();
  await expect(page.locator(".drawer")).toHaveCSS("background-color", "rgb(255, 255, 255)");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "历史记录" })).toBeHidden();
  await expect(historyTrigger).toBeFocused();

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
  const clipboard = page.getByRole("button", { name: "打开时读取剪贴板" });
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

test("keyboard shortcuts manage panels without conflicting with text actions", async ({ page }) => {
  await page.goto("/");

  await page.keyboard.press("ControlOrMeta+Shift+H");
  await expect(page.getByRole("dialog", { name: "历史记录" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "搜索翻译记录" })).toBeFocused();

  await page.keyboard.press("ControlOrMeta+L");
  await expect(page.getByRole("dialog", { name: "历史记录" })).toBeHidden();
  await expect(page.getByRole("textbox", { name: "原文" })).toBeFocused();

  await page.keyboard.press("ControlOrMeta+,");
  await expect(page.getByRole("dialog", { name: "偏好设置" })).toBeVisible();
  await page.keyboard.press("ControlOrMeta+,");
  await expect(page.getByRole("dialog", { name: "偏好设置" })).toBeHidden();
});

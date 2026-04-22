import { Browser, BrowserContext, Page } from "@playwright/test";

type Role = "author" | "reviewer" | "approver" | "admin";

function resolveBaseURL(): string {
  return process.env.E2E_BASE_URL || "http://localhost:8080";
}

function cookieDomainFromBaseURL(baseURL: string): string {
  return new URL(baseURL).hostname;
}

function cookieSecureFromBaseURL(baseURL: string): boolean {
  return new URL(baseURL).protocol === "https:";
}

export async function loginAs(
  page: Page,
  cookies: Record<string, string>,
  role: Role
): Promise<void> {
  const baseURL = resolveBaseURL();
  const cookieValue = cookies[role];
  if (!cookieValue) {
    throw new Error(`Missing session cookie for role '${role}'`);
  }

  await page.context().addCookies([
    {
      name: "metaldocs_session",
      value: cookieValue,
      domain: cookieDomainFromBaseURL(baseURL),
      path: "/",
      httpOnly: true,
      secure: cookieSecureFromBaseURL(baseURL),
      sameSite: "Lax",
    },
  ]);
}

export async function contextAs(
  browser: Browser,
  baseURL: string,
  cookies: Record<string, string>,
  role: Role
): Promise<BrowserContext> {
  const cookieValue = cookies[role];
  if (!cookieValue) {
    throw new Error(`Missing session cookie for role '${role}'`);
  }

  const context = await browser.newContext({ baseURL });
  await context.addCookies([
    {
      name: "metaldocs_session",
      value: cookieValue,
      domain: cookieDomainFromBaseURL(baseURL),
      path: "/",
      httpOnly: true,
      secure: cookieSecureFromBaseURL(baseURL),
      sameSite: "Lax",
    },
  ]);

  return context;
}

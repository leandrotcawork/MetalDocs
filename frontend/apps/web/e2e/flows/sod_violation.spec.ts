import { randomUUID } from 'node:crypto';

import { type APIResponse, type Page, type Request } from '@playwright/test';

import { test, expect } from '../fixtures/isolation';
import type { IsolatedFixture } from '../fixtures/isolation';
import { loginAs } from '../utils/auth';

type RoleCookies = Record<'author' | 'reviewer' | 'approver' | 'admin', string>;

type GovernanceEventRow = {
  event_type: string;
  payload_json?: Record<string, unknown>;
};

type SubmitContext = {
  routeId: string;
  instanceId: string;
};

const SOD_ERROR_CODE = 'sod.submitter_cannot_sign';
const INLINE_DIALOG_ERROR = 'Erro interno do servidor. Tente novamente em instantes.';
const PERMISSION_DENIED_TOAST = 'Permissão negada.';

function roleCookies(isolated: IsolatedFixture): RoleCookies {
  return {
    author: isolated.users.author.cookie,
    reviewer: isolated.users.reviewer.cookie,
    approver: isolated.users.approver.cookie,
    admin: isolated.users.admin.cookie,
  };
}

function stateBadge(page: Page) {
  return page.locator('[data-testid="state-badge"], [aria-label^="Estado:"]').first();
}

async function stateBadgeText(page: Page): Promise<string> {
  const locator = stateBadge(page);
  const ariaLabel = await locator.getAttribute('aria-label');
  if (ariaLabel?.startsWith('Estado:')) {
    return ariaLabel.slice('Estado:'.length).trim();
  }
  const rawText = (await locator.textContent()) ?? '';
  return rawText.replace(/\s+/g, ' ').trim();
}

function signoffRequest(request: Request): boolean {
  return request.method() === 'POST' && request.url().includes('/signoff');
}

async function submitAsAuthor(page: Page, isolated: IsolatedFixture): Promise<SubmitContext> {
  const cookies = roleCookies(isolated);
  await loginAs(page, cookies, 'author');
  await page.goto(`/documents/${isolated.docId}`);

  const submitRequestPromise = page.waitForRequest((request) => {
    return request.method() === 'POST' && request.url().includes(`/api/v2/documents/${isolated.docId}/submit`);
  });
  const submitResponsePromise = page.waitForResponse((response) => {
    return response.request().method() === 'POST' && response.url().includes(`/api/v2/documents/${isolated.docId}/submit`);
  });

  await page.getByRole('button', { name: 'Submeter para revisão' }).click();
  await page.getByRole('button', { name: /^Submeter$/ }).click();

  const submitRequest = await submitRequestPromise;
  const submitResponse = await submitResponsePromise;
  expect(submitResponse.ok()).toBeTruthy();

  const submitBody = submitRequest.postDataJSON() as { route_id?: string };
  const submitRespBody = (await submitResponse.json()) as { instance_id?: string };

  expect(submitBody.route_id).toBeTruthy();
  expect(submitRespBody.instance_id).toBeTruthy();

  await expect.poll(() => stateBadgeText(page), { timeout: 10_000 }).toContain('Em revis');

  return {
    routeId: submitBody.route_id as string,
    instanceId: submitRespBody.instance_id as string,
  };
}

async function governanceEvents(page: Page, tenantId: string, docId: string): Promise<GovernanceEventRow[]> {
  const query = new URLSearchParams({ tenantId, docId });
  const response = await page.request.get(`/internal/test/governance-events?${query.toString()}`);
  expect(response.ok()).toBeTruthy();
  return (await response.json()) as GovernanceEventRow[];
}

function firstStageId(events: GovernanceEventRow[], instanceId: string): string | null {
  const asString = (value: unknown): string | null => {
    return typeof value === 'string' ? value : null;
  };

  const stageEvent = events.find((event) => {
    const eventType = event.event_type.toLowerCase();
    return eventType.includes('stage') && eventType.includes('activ');
  });

  const payload = stageEvent?.payload_json;
  const payloadStageId = asString(payload?.['stage_instance_id']) ?? asString(payload?.['stage_id']);
  if (payloadStageId) {
    return payloadStageId;
  }

  const fallback = events.find((event) => {
    const payloadInstance = event.payload_json?.['instance_id'];
    return typeof payloadInstance === 'string' && payloadInstance === instanceId;
  });
  const fallbackPayload = fallback?.payload_json;
  const fallbackStageId =
    asString(fallbackPayload?.['stage_instance_id']) ?? asString(fallbackPayload?.['stage_id']);
  if (fallbackStageId) return fallbackStageId;

  return null;
}

async function addAuthorAsStageMember(params: {
  page: Page;
  routeId: string;
  stageId: string | null;
  authorUserId: string;
  adminCookie: string;
}): Promise<void> {
  const { page, routeId, stageId, authorUserId, adminCookie } = params;

  const candidateURLs: string[] = [];
  if (stageId) {
    candidateURLs.push(`/api/v2/routes/${routeId}/stages/${stageId}/members`);
    candidateURLs.push(`/api/v2/approval/routes/${routeId}/stages/${stageId}/members`);
  }
  candidateURLs.push(`/api/v2/routes/${routeId}/stages/1/members`);
  candidateURLs.push(`/api/v2/approval/routes/${routeId}/stages/1/members`);

  const payloads: Array<Record<string, unknown>> = [
    { user_id: authorUserId },
    { member_user_id: authorUserId },
    { actor_user_id: authorUserId },
    { member_id: authorUserId },
    { userId: authorUserId },
  ];

  let lastError = '';

  for (const url of candidateURLs) {
    for (const data of payloads) {
      const response = await page.request.post(url, {
        headers: {
          Cookie: `metaldocs_session=${adminCookie}`,
          'Idempotency-Key': randomUUID(),
        },
        data,
      });

      if ([200, 201, 204, 409].includes(response.status())) {
        return;
      }

      if ([404, 405].includes(response.status())) {
        continue;
      }

      const bodyText = await response.text().catch(() => '');
      lastError = `POST ${url} -> ${response.status()} ${bodyText}`;
      break;
    }
  }

  throw new Error(lastError || 'Unable to add author as stage member: endpoint not found');
}

async function attemptAuthorSignoffFromInbox(
  page: Page,
  isolated: IsolatedFixture,
): Promise<{ response: APIResponse; code: string | null }> {
  const cookies = roleCookies(isolated);
  await loginAs(page, cookies, 'author');
  await page.goto('/approval/inbox');

  const row = page.locator('tbody tr').filter({ hasText: isolated.docId }).first();
  await expect(row).toBeVisible();
  await row.click();

  await page.getByRole('button', { name: /Assinar/i }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();

  await page.getByLabel('Senha').fill('test1234');

  const signoffResponsePromise = page.waitForResponse((response) => signoffRequest(response.request()));
  await page.getByRole('button', { name: /Aprovar|Confirmar assinatura/i }).click();

  const response = await signoffResponsePromise;
  const body = (await response.json().catch(() => ({}))) as {
    code?: string;
    error?: { code?: string };
  };

  return {
    response,
    code: body.code ?? body.error?.code ?? null,
  };
}

test.describe.serial('sod_violation', () => {
  test('author submits, then cannot sign own doc', async ({ page, isolated }) => {
    const submit = await submitAsAuthor(page, isolated);
    const eventsAfterSubmit = await governanceEvents(page, isolated.tenantId, isolated.docId);
    const stageId = firstStageId(eventsAfterSubmit, submit.instanceId);

    await addAuthorAsStageMember({
      page,
      routeId: submit.routeId,
      stageId,
      authorUserId: isolated.users.author.id,
      adminCookie: isolated.users.admin.cookie,
    });

    const { response, code } = await attemptAuthorSignoffFromInbox(page, isolated);

    expect(response.status()).toBe(403);
    expect(code).toBe(SOD_ERROR_CODE);

    const toast = page
      .locator('[data-sonner-toast], [data-testid="app-toast-error"], [role="alert"]')
      .filter({ hasText: PERMISSION_DENIED_TOAST })
      .first();
    await expect(toast).toBeVisible();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText(INLINE_DIALOG_ERROR)).toBeVisible();
  });

  test('governance events: no signoff row, one signoff_denied_sod event', async ({ page, isolated }) => {
    const submit = await submitAsAuthor(page, isolated);
    const eventsAfterSubmit = await governanceEvents(page, isolated.tenantId, isolated.docId);
    const stageId = firstStageId(eventsAfterSubmit, submit.instanceId);

    await addAuthorAsStageMember({
      page,
      routeId: submit.routeId,
      stageId,
      authorUserId: isolated.users.author.id,
      adminCookie: isolated.users.admin.cookie,
    });

    const { response, code } = await attemptAuthorSignoffFromInbox(page, isolated);
    expect(response.status()).toBe(403);
    expect(code).toBe(SOD_ERROR_CODE);

    const events = await governanceEvents(page, isolated.tenantId, isolated.docId);
    const eventTypes = events.map((event) => event.event_type);

    expect(eventTypes.filter((type) => type === 'signoff.recorded')).toHaveLength(0);
    expect(
      eventTypes.filter((type) => type === 'signoff.denied.sod' || type === 'approval.signoff.denied_sod'),
    ).toHaveLength(1);
  });

  test('non-member reviewer cannot see doc in inbox', async ({ page, isolated }) => {
    await submitAsAuthor(page, isolated);

    const cookies = roleCookies(isolated);
    await loginAs(page, cookies, 'reviewer');
    await page.goto('/approval/inbox');

    const row = page.locator('tbody tr').filter({ hasText: isolated.docId });
    await expect.poll(async () => row.count(), { timeout: 5_000 }).toBe(0);
  });

  test('cross-tenant: reviewer from tenant B cannot sign tenant A\'s doc', async ({ page, isolated }) => {
    await submitAsAuthor(page, isolated);

    const tenantBId = `e2e_cross_${randomUUID().replace(/-/g, '').slice(0, 12)}`;

    const seedTenantB = await page.request.post('/internal/test/seed', {
      data: {
        tenantId: tenantBId,
        docId: randomUUID(),
        roles: ['author', 'reviewer', 'approver', 'admin'],
      },
    });
    expect(seedTenantB.ok()).toBeTruthy();

    try {
      const tenantB = (await seedTenantB.json()) as {
        cookies: {
          reviewer: string;
        };
      };

      const crossTenantResponse = await page.request.post(`/api/v2/documents/${isolated.docId}/signoff`, {
        headers: {
          Cookie: `metaldocs_session=${tenantB.cookies.reviewer}`,
          'Idempotency-Key': randomUUID(),
        },
        data: {
          decision: 'approve',
          password: 'test1234',
          content_hash: 'e2e-cross-tenant-attempt',
        },
      });

      expect([403, 404]).toContain(crossTenantResponse.status());
    } finally {
      await page.request.post('/internal/test/reset', { data: { tenantId: tenantBId } });
    }
  });
});

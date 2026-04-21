import { lazy, Suspense, useState } from 'react';
import { TemplateCreateDialog } from './TemplateCreateDialog';

const TemplatesListPage = lazy(() => import('./TemplatesListPage').then(m => ({ default: m.TemplatesListPage })));
const TemplateAuthorPage = lazy(() => import('./TemplateAuthorPage').then(m => ({ default: m.TemplateAuthorPage })));

export type TemplatesV2Route =
  | { kind: 'list' }
  | { kind: 'author'; templateId: string; versionNum: number };

export function TemplatesV2View({
  route,
  onNavigate,
}: {
  route: TemplatesV2Route;
  onNavigate: (next: TemplatesV2Route) => void;
}) {
  const [showCreate, setShowCreate] = useState(false);

  if (route.kind === 'list') {
    return (
      <Suspense fallback={<div>Loading…</div>}>
        <TemplatesListPage
          onOpenTemplate={(templateId, versionNum) => onNavigate({ kind: 'author', templateId, versionNum })}
          onCreate={() => setShowCreate(true)}
        />
        {showCreate && (
          <TemplateCreateDialog
            onClose={() => setShowCreate(false)}
            onCreated={(templateId, versionNum) => {
              setShowCreate(false);
              onNavigate({ kind: 'author', templateId, versionNum });
            }}
          />
        )}
      </Suspense>
    );
  }

  return (
    <Suspense fallback={<div>Loading…</div>}>
      <TemplateAuthorPage
        templateId={route.templateId}
        versionNum={route.versionNum}
        onNavigateToVersion={(templateId, versionNum) => onNavigate({ kind: 'author', templateId, versionNum })}
        onBack={() => onNavigate({ kind: 'list' })}
      />
    </Suspense>
  );
}

import { DocumentCreatePage } from './DocumentCreatePage';
import { DocumentEditorPage } from './DocumentEditorPage';

export type DocumentsV2Route =
  | { kind: 'create' }
  | { kind: 'editor'; documentID: string };

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function routeFromPath(pathname: string): DocumentsV2Route {
  if (pathname === '/documents-v2' || pathname === '/documents-v2/' || pathname === '/documents-v2/new') {
    return { kind: 'create' };
  }
  if (pathname.startsWith('/documents-v2/')) {
    const id = pathname.slice('/documents-v2/'.length).replace(/\/+$/, '');
    if (UUID_RE.test(id)) {
      return { kind: 'editor', documentID: id };
    }
  }
  return { kind: 'create' };
}

export function pathFromRoute(route: DocumentsV2Route): string {
  if (route.kind === 'create') return '/documents-v2/new';
  return `/documents-v2/${route.documentID}`;
}

export function renderDocumentsV2View(
  route: DocumentsV2Route,
  onNavigate: (next: DocumentsV2Route) => void,
): React.ReactElement {
  if (route.kind === 'create') {
    return (
      <DocumentCreatePage
        onCreated={(documentID) => {
          onNavigate({ kind: 'editor', documentID });
        }}
      />
    );
  }

  return (
    <DocumentEditorPage
      documentID={route.documentID}
      onDone={() => {
        onNavigate({ kind: 'create' });
      }}
    />
  );
}

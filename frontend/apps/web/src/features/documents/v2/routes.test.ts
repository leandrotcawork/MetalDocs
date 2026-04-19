import { describe, expect, it, vi } from 'vitest';

vi.mock('@metaldocs/editor-ui', () => ({ MetalDocsEditor: () => null }));

import { pathFromRoute, routeFromPath } from './routes';

describe('documents-v2 routeFromPath', () => {
  it('maps base create paths to create route', () => {
    expect(routeFromPath('/documents-v2')).toEqual({ kind: 'create' });
    expect(routeFromPath('/documents-v2/')).toEqual({ kind: 'create' });
    expect(routeFromPath('/documents-v2/new')).toEqual({ kind: 'create' });
  });

  it('maps uuid path to editor route', () => {
    expect(routeFromPath('/documents-v2/123e4567-e89b-12d3-a456-426614174000')).toEqual({
      kind: 'editor',
      documentID: '123e4567-e89b-12d3-a456-426614174000',
    });
  });

  it('maps non-uuid path to create route', () => {
    expect(routeFromPath('/documents-v2/not-a-uuid')).toEqual({ kind: 'create' });
  });
});

describe('documents-v2 pathFromRoute', () => {
  it('round-trips create route', () => {
    const route = routeFromPath(pathFromRoute({ kind: 'create' }));
    expect(route).toEqual({ kind: 'create' });
  });

  it('round-trips editor route', () => {
    const original = { kind: 'editor', documentID: '123e4567-e89b-12d3-a456-426614174000' } as const;
    const route = routeFromPath(pathFromRoute(original));
    expect(route).toEqual(original);
  });
});

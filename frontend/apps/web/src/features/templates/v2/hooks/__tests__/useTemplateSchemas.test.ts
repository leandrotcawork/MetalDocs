import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useTemplateSchemas } from '../useTemplateSchemas';
import * as api from '../../api/templatesV2';

vi.mock('../../api/templatesV2');

const schemas = {
  placeholders: [{ id: 'p1', label: 'Title', type: 'text' as const }],
  zones: [],
  composition: null,
};

describe('useTemplateSchemas', () => {
  beforeEach(() => {
    vi.mocked(api.getTemplateSchemas).mockResolvedValue(schemas);
    vi.mocked(api.putTemplateSchemas).mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads schemas on mount', async () => {
    const { result } = renderHook(() => useTemplateSchemas('t1', 1));
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.schemas).toEqual(schemas);
    expect(result.current.error).toBeNull();
  });

  it('sets error on load failure', async () => {
    vi.mocked(api.getTemplateSchemas).mockRejectedValue(new Error('not found'));
    const { result } = renderHook(() => useTemplateSchemas('t1', 1));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe('not found');
    expect(result.current.schemas).toBeNull();
  });

  it('save calls putTemplateSchemas and updates local state', async () => {
    const { result } = renderHook(() => useTemplateSchemas('t1', 1));
    await waitFor(() => expect(result.current.loading).toBe(false));

    const updated = { ...schemas, placeholders: [] };
    await act(async () => {
      await result.current.save(updated);
    });

    expect(vi.mocked(api.putTemplateSchemas)).toHaveBeenCalledWith('t1', 1, updated);
    expect(result.current.schemas).toEqual(updated);
  });

  it('saving flag is true during save', async () => {
    let resolve!: () => void;
    vi.mocked(api.putTemplateSchemas).mockReturnValue(new Promise<void>((r) => { resolve = r; }));

    const { result } = renderHook(() => useTemplateSchemas('t1', 1));
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => { void result.current.save(schemas); });
    expect(result.current.saving).toBe(true);

    await act(async () => { resolve(); });
    expect(result.current.saving).toBe(false);
  });
});

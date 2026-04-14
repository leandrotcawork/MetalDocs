import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  getTemplate,
  editPublished,
  saveDraft as apiSaveDraft,
  publishTemplate,
  discardDraft as apiDiscardDraft,
  TemplateLockConflictError,
  TemplatePublishValidationError,
} from "../../api/templates";
import type { TemplateDraftDTO } from "../../api/templates";
import { useTemplatesStore } from "../../store/templates.store";
import { validateTemplate } from "../documents/mddm-editor/engine/codecs/validate-template";

// NOTE: The project has no generic toast library (only the operations-stream
// notification system). For conflict alerts we fall back to window.alert().
// Replace with a proper toast when one is wired in.

interface UseTemplateDraftOptions {
  templateKey: string;
}

interface UseTemplateDraftResult {
  draft: TemplateDraftDTO | null;
  isLoading: boolean;
  error: string | null;
  saveDraft: (blocks: unknown) => Promise<void>;
  publish: (blocks: unknown) => Promise<void>;
  discardDraft: () => Promise<void>;
  replaceDraft: (draft: TemplateDraftDTO) => void;
}

export function useTemplateDraft({ templateKey }: UseTemplateDraftOptions): UseTemplateDraftResult {
  const navigate = useNavigate();
  const { setDraft, setActiveTemplate, setValidationErrors, markClean } = useTemplatesStore();

  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [localDraft, setLocalDraft] = useState<TemplateDraftDTO | null>(null);

  // Load the template on mount
  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      setError(null);
      try {
        const result = await getTemplate(templateKey);
        if (cancelled) return;

        const draft = (result as TemplateDraftDTO).status === "draft" && "lockVersion" in (result as TemplateDraftDTO)
          ? (result as TemplateDraftDTO)
          : await editPublished(templateKey);

        if (cancelled) return;
        setLocalDraft(draft);
        setDraft(draft);
        setActiveTemplate(templateKey);
        markClean();
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : "Erro ao carregar template.");
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    }

    void load();
    return () => { cancelled = true; };
  }, [templateKey, setDraft, setActiveTemplate, markClean]);

  const saveDraft = useCallback(
    async (blocks: unknown) => {
      const current = localDraft;
      if (!current) return;

      try {
        const updated = await apiSaveDraft(templateKey, {
          blocks,
          lockVersion: current.lockVersion,
        });
        setLocalDraft(updated);
        setDraft(updated);
        markClean();
      } catch (err) {
        if (err instanceof TemplateLockConflictError) {
          // eslint-disable-next-line no-alert
          alert(
            "Conflito de edicao: o template foi modificado por outra sessao. " +
            "Recarregue a pagina para obter a versao mais recente."
          );
          return;
        }
        throw err;
      }
    },
    [templateKey, localDraft, setDraft, markClean],
  );

  const publish = useCallback(
    async (blocks: unknown) => {
      const current = localDraft;
      if (!current) return;

      // Client-side validation before hitting the server
      const blocksArray = Array.isArray(blocks) ? blocks : [];
      const clientErrors = validateTemplate(blocksArray);
      if (clientErrors.length > 0) {
        setValidationErrors(clientErrors);
        return;
      }

      try {
        const savedDraft = await apiSaveDraft(templateKey, {
          blocks,
          lockVersion: current.lockVersion,
        });
        setLocalDraft(savedDraft);
        setDraft(savedDraft);
        markClean();

        await publishTemplate(templateKey, savedDraft.lockVersion);
        // On success, clear validation errors and navigate back to profile list
        setValidationErrors([]);
        navigate(-1);
      } catch (err) {
        if (err instanceof TemplateLockConflictError) {
          // eslint-disable-next-line no-alert
          alert(
            "Conflito de edicao ao publicar: o template foi modificado por outra sessao. " +
            "Recarregue a pagina."
          );
          return;
        }
        if (err instanceof TemplatePublishValidationError) {
          setValidationErrors(err.errors);
          return;
        }
        throw err;
      }
    },
    [templateKey, localDraft, setValidationErrors, navigate],
  );

  const discardDraft = useCallback(async () => {
    try {
      await apiDiscardDraft(templateKey);
      navigate(-1);
    } catch (err) {
      // Re-throw so the caller (MetadataBar confirm dialog) can surface it
      throw err;
    }
  }, [templateKey, navigate]);

  const replaceDraft = useCallback((nextDraft: TemplateDraftDTO) => {
    setLocalDraft(nextDraft);
    setDraft(nextDraft);
    markClean();
  }, [setDraft, markClean]);

  return {
    draft: localDraft,
    isLoading,
    error,
    saveDraft,
    publish,
    discardDraft,
    replaceDraft,
  };
}

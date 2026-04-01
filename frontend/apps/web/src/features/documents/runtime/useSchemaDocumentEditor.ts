import { useCallback, useEffect, useRef } from "react";
import { fetchDocumentEditorBundle, fetchDocumentTypeBundle, saveDocumentContent } from "../../../api/documents";
import { useDocumentsStore } from "../../../store/documents.store";
import { createSchemaDocumentEditorState, normalizeSchemaDocumentEditorBundle } from "./schemaRuntimeAdapters";

type UseSchemaDocumentEditorOptions = {
  documentId?: string | null;
  typeKey?: string | null;
  autoLoad?: boolean;
  initialValues?: Record<string, unknown>;
};

export function useSchemaDocumentEditor(options: UseSchemaDocumentEditorOptions = {}) {
  const editor = useDocumentsStore((state) => state.schemaDocumentEditor);
  const selectedDocumentTypeKey = useDocumentsStore((state) => state.selectedDocumentTypeKey);
  const setSchemaDocumentEditor = useDocumentsStore((state) => state.setSchemaDocumentEditor);
  const setSelectedDocumentTypeKey = useDocumentsStore((state) => state.setSelectedDocumentTypeKey);
  const initialValuesRef = useRef<Record<string, unknown>>(options.initialValues ?? {});

  useEffect(() => {
    const nextTypeKey = options.typeKey ?? "";
    if (nextTypeKey !== selectedDocumentTypeKey) {
      setSelectedDocumentTypeKey(nextTypeKey);
    }
  }, [options.typeKey, selectedDocumentTypeKey, setSelectedDocumentTypeKey]);

  const loadEditor = useCallback(async (isActive: () => boolean) => {
    const documentId = options.documentId ?? "";
    const typeKey = options.typeKey ?? "";

    if (documentId) {
      setSchemaDocumentEditor(
        createSchemaDocumentEditorState({
          documentId,
          typeKey,
          status: "loading",
        }),
      );

      try {
        const bundle = normalizeSchemaDocumentEditorBundle(await fetchDocumentEditorBundle(documentId));
        if (!isActive()) {
          return;
        }
        setSchemaDocumentEditor(
          createSchemaDocumentEditorState({
            documentId: bundle.document.documentId,
            typeKey: bundle.typeKey || typeKey,
            schema: bundle.schema,
            values: bundle.values,
            version: bundle.version,
            pdfUrl: bundle.pdfUrl,
            status: "idle",
            bundle: null,
            document: bundle.document,
          }),
        );
      } catch (error) {
        if (!isActive()) {
          return;
        }
        setSchemaDocumentEditor(
          createSchemaDocumentEditorState({
            documentId,
            typeKey,
            status: "error",
            error: error instanceof Error ? error.message : "Falha ao carregar o editor de schema.",
          }),
        );
      }
      return;
    }

    if (!typeKey) {
      setSchemaDocumentEditor(
        createSchemaDocumentEditorState({
          values: initialValuesRef.current,
        }),
      );
      return;
    }

    setSchemaDocumentEditor(
      createSchemaDocumentEditorState({
        documentId: "",
        typeKey,
        values: initialValuesRef.current,
        status: "loading",
      }),
    );

    try {
      const bundle = await fetchDocumentTypeBundle(typeKey);
      if (!isActive()) {
        return;
      }
      setSchemaDocumentEditor(
        createSchemaDocumentEditorState({
          documentId: "",
          typeKey: bundle.typeKey || typeKey,
          schema: bundle.schema,
          values: initialValuesRef.current,
          status: "idle",
          bundle,
        }),
      );
    } catch (error) {
      if (!isActive()) {
        return;
      }
      setSchemaDocumentEditor(
        createSchemaDocumentEditorState({
          documentId: "",
          typeKey,
          values: initialValuesRef.current,
          status: "error",
          error: error instanceof Error ? error.message : "Falha ao carregar o schema do documento.",
        }),
      );
    }
  }, [options.documentId, options.typeKey, setSchemaDocumentEditor]);

  useEffect(() => {
    if (options.autoLoad === false) {
      return;
    }

    let active = true;
    void loadEditor(() => active);

    return () => {
      active = false;
    };
  }, [loadEditor, options.autoLoad]);

  const setValues = useCallback(
    (nextValues: Record<string, unknown> | ((current: Record<string, unknown>) => Record<string, unknown>)) => {
      setSchemaDocumentEditor((current) => {
        const values = typeof nextValues === "function" ? nextValues(current.values) : nextValues;
        return {
          ...current,
          values,
        };
      });
    },
    [setSchemaDocumentEditor],
  );

  const save = useCallback(
    async (nextValues?: Record<string, unknown>) => {
      const documentId = options.documentId ?? editor.documentId;
      if (!documentId) {
        throw new Error("Documento nao identificado para salvar o schema runtime.");
      }

      const values = nextValues ?? editor.values;
      setSchemaDocumentEditor((current) => ({ ...current, status: "saving", error: "" }));
      try {
        const response = await saveDocumentContent(documentId, values);
        setSchemaDocumentEditor((current) => ({
          ...current,
          values,
          version: response.version ?? current.version,
          pdfUrl: response.pdfUrl ?? current.pdfUrl,
          status: "idle",
        }));
        return response;
      } catch (error) {
        setSchemaDocumentEditor((current) => ({
          ...current,
          status: "error",
          error: error instanceof Error ? error.message : "Falha ao salvar o schema runtime.",
        }));
        throw error;
      }
    },
    [editor.documentId, editor.values, options.documentId, setSchemaDocumentEditor],
  );

  return {
    editor,
    selectedDocumentTypeKey,
    setValues,
    save,
    refresh: () => loadEditor(() => true),
  };
}

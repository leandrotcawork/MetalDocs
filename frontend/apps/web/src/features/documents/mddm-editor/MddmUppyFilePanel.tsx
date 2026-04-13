import Uppy from "@uppy/core";
import Dashboard from "@uppy/react/dashboard";
import XHR from "@uppy/xhr-upload";
import ImageEditor from "@uppy/image-editor";
import { useBlockNoteEditor } from "@blocknote/react";
import type { FilePanelProps } from "@blocknote/react";
import { useEffect, useMemo } from "react";
import "@uppy/core/css/style.min.css";
import "@uppy/dashboard/css/style.min.css";
import "@uppy/image-editor/css/style.min.css";
import { API_BASE_URL } from "../../../api/client";

export type MddmUppyFilePanelProps = FilePanelProps & {
  documentId: string;
};

export function MddmUppyFilePanel({ blockId, documentId }: MddmUppyFilePanelProps) {
  const editor = useBlockNoteEditor();

  const uppy = useMemo(
    () =>
      new Uppy({
        restrictions: { maxNumberOfFiles: 1, allowedFileTypes: ["image/*"] },
        autoProceed: false,
      })
        .use(XHR, {
          endpoint: `${API_BASE_URL}/documents/${documentId}/attachments`,
          method: "POST",
          fieldName: "file",
          withCredentials: true,
          getResponseData(xhr) {
            try {
              const { attachmentId } = JSON.parse(xhr.responseText) as { attachmentId: string };
              return {
                url: `/api/v1/documents/${documentId}/attachments/${attachmentId}/download-url`,
              };
            } catch {
              return {};
            }
          },
        })
        .use(ImageEditor, { quality: 0.85 }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [documentId],
  );

  // Wire upload-success → set image URL on the block
  useEffect(() => {
    function onUploadSuccess(
      _file: Parameters<Parameters<typeof uppy.on<"upload-success">>[1]>[0],
      response: Parameters<Parameters<typeof uppy.on<"upload-success">>[1]>[1],
    ) {
      const url = response.uploadURL ?? (response.body as Record<string, string> | undefined)?.url;
      if (url) {
        editor.updateBlock(blockId, {
          type: "image",
          props: { url },
        } as Parameters<typeof editor.updateBlock>[1]);
      }
    }

    uppy.on("upload-success", onUploadSuccess);
    return () => {
      uppy.off("upload-success", onUploadSuccess);
    };
  }, [uppy, blockId, editor]);

  // Destroy Uppy instance when component unmounts
  useEffect(() => () => void uppy.destroy(), [uppy]);

  return (
    <Dashboard
      uppy={uppy}
      plugins={["ImageEditor"]}
      proudlyDisplayPoweredByUppy={false}
      height={380}
      width={750}
      theme="light"
    />
  );
}

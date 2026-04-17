import JSZip from "jszip"
import { Packer, type Document } from "docx"

export async function readDocxDocumentXml(doc: Document): Promise<string> {
  const buf = await Packer.toBuffer(doc)
  const zip = await JSZip.loadAsync(buf)
  const entry = zip.file("word/document.xml")
  if (!entry) {
    throw new Error("word/document.xml missing from generated .docx")
  }
  return entry.async("string")
}

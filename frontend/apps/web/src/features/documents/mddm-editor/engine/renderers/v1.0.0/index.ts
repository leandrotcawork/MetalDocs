// Renderer Bundle: v1.0.0
//
// Captures the launch version of the MDDM engine. At v1.0.0, the bundle
// re-exports the live modules from engine/docx-emitter, engine/layout-ir,
// and engine/print-stylesheet. When the next renderer version is introduced
// (v1.1.0), this file is frozen: replace the re-exports with a local copy of
// the implementation to prevent the current-HEAD modules from drifting away
// from the v1.0.0 pin.

export { defaultLayoutTokens, defaultComponentRules } from "../../layout-ir";
export { mddmToDocx, MissingEmitterError, type EmitContext } from "../../docx-emitter";
export { PRINT_STYLESHEET, wrapInPrintDocument } from "../../print-stylesheet";

// Layout IR hash captured at the moment v1.0.0 was cut. This MUST match
// the value written in engine/ir-hash/recorded-hash.ts at the time of
// tagging. The drift gate in Part 4 enforces this.
import { RECORDED_IR_HASH, RECORDED_RENDERER_VERSION } from "../../ir-hash/recorded-hash";
export const BUNDLE_RENDERER_VERSION = RECORDED_RENDERER_VERSION;
export const BUNDLE_LAYOUT_IR_HASH = RECORDED_IR_HASH;

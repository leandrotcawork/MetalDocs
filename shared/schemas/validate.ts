import Ajv from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import { readFileSync } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const schemaPath = join(__dirname, "mddm.schema.json");
const schema = JSON.parse(readFileSync(schemaPath, "utf8"));

const ajv = new Ajv({ allErrors: true, strict: false });
addFormats(ajv);
const validateFn = ajv.compile(schema);

export type MDDMValidationResult = {
  valid: boolean;
  errors?: Array<{ path: string; message: string }>;
};

export function validateMDDM(envelope: unknown): MDDMValidationResult {
  const valid = validateFn(envelope);
  if (valid) return { valid: true };
  return {
    valid: false,
    errors: (validateFn.errors ?? []).map((e) => ({
      path: e.instancePath,
      message: e.message ?? "validation error",
    })),
  };
}

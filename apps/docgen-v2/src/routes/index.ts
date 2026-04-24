import type { FastifyInstance } from 'fastify';
import type { Env } from '../env.js';
import type { Client } from 'minio';
import { registerValidateTemplate } from './validate-template.js';
import { registerRenderRoutes } from './render.js';
import { registerConvertPDF } from './convert-pdf.js';
import { registerFanoutRoute } from './fanout.js';

export function registerRoutes(app: FastifyInstance, env: Env, s3Factory: () => Client): void {
  registerValidateTemplate(app, env, s3Factory);
  registerRenderRoutes(app, env, s3Factory);
  registerConvertPDF(app, env, s3Factory);
  registerFanoutRoute(app, env, s3Factory);
}

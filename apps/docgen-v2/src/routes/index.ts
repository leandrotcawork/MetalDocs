import type { FastifyInstance } from 'fastify';
import type { Env } from '../env.js';
import type { Client } from 'minio';
import { registerValidateTemplate } from './validate-template.js';

export function registerRoutes(app: FastifyInstance, env: Env, s3Factory: () => Client): void {
  registerValidateTemplate(app, env, s3Factory);
}

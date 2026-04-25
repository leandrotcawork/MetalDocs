export interface SubBlockContext {
  params: Record<string, unknown>;
  values: Record<string, unknown>;
}

export interface SubBlockRenderer {
  key: string;
  render(ctx: SubBlockContext): Promise<string>;
}

export class SubBlockRegistry {
  private map = new Map<string, SubBlockRenderer>();

  register(r: SubBlockRenderer): void {
    this.map.set(r.key, r);
  }

  async render(key: string, ctx: SubBlockContext): Promise<string> {
    const r = this.map.get(key);
    if (!r) throw new Error(`unknown sub-block: ${key}`);
    return r.render(ctx);
  }

  keys(): string[] {
    return Array.from(this.map.keys());
  }
}

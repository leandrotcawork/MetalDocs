export function asMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Falha inesperada.";
}

export function statusOf(error: unknown): number | undefined {
  if (error && typeof error === "object" && "status" in error && typeof (error as { status?: unknown }).status === "number") {
    return (error as { status: number }).status;
  }
  return undefined;
}

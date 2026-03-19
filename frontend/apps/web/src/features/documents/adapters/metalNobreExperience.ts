import type { DocumentProfileItem, ProcessAreaItem } from "../../../lib.types";

const profileContextByCode: Record<string, string> = {
  po: "Padrao para procedimentos do SGQ com foco em fluxos operacionais e rastreabilidade.",
  it: "Instrucao orientada a execucao de tarefa, com passo a passo e criterios de aceite.",
  rg: "Registro de evidencia operacional para auditoria, retencao e conformidade ISO-inspired.",
};

const processAreaHintByCode: Record<string, string> = {
  quality: "Controles de qualidade, auditorias internas e evidencias do SGQ.",
  marketplaces: "Rotinas operacionais e comerciais dos canais de marketplace.",
  commercial: "Processos comerciais, propostas e relacionamento com cliente.",
  purchasing: "Aquisicao de materiais, homologacao e relacionamento com fornecedores.",
  logistics: "Movimentacao, expedicao, recebimento e conformidade logistica.",
  finance: "Controles financeiros, fiscais e conciliacao de evidencias.",
};

export function metalNobreProfileContext(profileCode: string): string {
  const key = profileCode.trim().toLowerCase();
  return profileContextByCode[key] ?? "Perfil documental configurado pelo registry com governanca ativa.";
}

export function metalNobreProcessAreaHint(processAreaCode: string): string {
  const key = processAreaCode.trim().toLowerCase();
  return processAreaHintByCode[key] ?? "Area operacional configurada no registry documental.";
}

export function metalNobreProfileOptionLabel(profile: DocumentProfileItem): string {
  return `${profile.name} (${profile.alias})`;
}

export function metalNobreProcessAreaOptionLabel(area: ProcessAreaItem): string {
  return area.name;
}

import type { TemplateDefinition } from "../engine/template";

export const poStandardTemplate: TemplateDefinition = {
  templateKey: "po-standard",
  version: 1,
  profileCode: "po",
  status: "published",
  meta: {
    name: "Procedimento Operacional Padrão",
    description: "Template padrão para procedimentos operacionais",
    createdAt: "2026-04-13T00:00:00Z",
    updatedAt: "2026-04-13T00:00:00Z",
  },
  theme: {
    accent: "#6b1f2a",
    accentLight: "#f9f3f3",
    accentDark: "#3e1018",
    accentBorder: "#dfc8c8",
  },
  blocks: [
    {
      type: "section",
      props: { title: "IDENTIFICAÇÃO" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "IDENTIFICAÇÃO DO PROCESSO" },
      capabilities: { locked: true, removable: false },
      children: [
        { type: "richBlock", props: { label: "Objetivo" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Escopo" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Cargo responsável" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Canal / Contexto" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Participantes" }, capabilities: { locked: true, editableZones: ["content"] } },
      ],
    },
    {
      type: "section",
      props: { title: "ENTRADAS E SAÍDAS" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "VISÃO GERAL DO PROCESSO" },
      capabilities: { locked: true, removable: false },
      children: [
        { type: "richBlock", props: { label: "Descrição do processo" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Diagrama" }, capabilities: { locked: true, editableZones: ["content"] } },
      ],
    },
    {
      type: "section",
      props: { title: "DETALHAMENTO DAS ETAPAS" },
      capabilities: { locked: true, removable: false },
      children: [
        {
          type: "repeatable",
          props: { label: "Etapas", itemPrefix: "Etapa" },
          capabilities: { locked: false, addItems: true, removeItems: true, maxItems: 50, minItems: 1 },
          children: [],
        },
      ],
    },
    {
      type: "section",
      props: { title: "INDICADORES" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "RISCOS E CONTROLES" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "REFERÊNCIAS" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "GLOSSÁRIO" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "HISTÓRICO DE REVISÕES" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
  ],
};

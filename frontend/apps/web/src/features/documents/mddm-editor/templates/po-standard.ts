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
      props: { title: "IDENTIFICAÇÃO", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "IDENTIFICAÇÃO DO PROCESSO", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [
        { type: "richBlock", props: { label: "Objetivo", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Escopo", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Cargo responsável", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Canal / Contexto", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Participantes", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
      ],
    },
    {
      type: "section",
      props: { title: "ENTRADAS E SAÍDAS", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "VISÃO GERAL DO PROCESSO", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [
        { type: "richBlock", props: { label: "Descrição do processo", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Diagrama", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
      ],
    },
    {
      type: "section",
      props: { title: "DETALHAMENTO DAS ETAPAS", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [
        {
          type: "repeatable",
          props: { label: "Etapas", itemPrefix: "Etapa", locked: false, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: false, addItems: true, removeItems: true, maxItems: 50, minItems: 1 }) },
          children: [],
        },
      ],
    },
    {
      type: "section",
      props: { title: "INDICADORES", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "RISCOS E CONTROLES", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "REFERÊNCIAS", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "GLOSSÁRIO", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "HISTÓRICO DE REVISÕES", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
  ],
};
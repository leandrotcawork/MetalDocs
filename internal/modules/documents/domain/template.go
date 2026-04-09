package domain

import (
	"strings"
	"time"
)

type TemplateExportConfig struct {
	MarginTop    float64 `json:"marginTop"`
	MarginRight  float64 `json:"marginRight"`
	MarginBottom float64 `json:"marginBottom"`
	MarginLeft   float64 `json:"marginLeft"`
}

type DocumentTemplateVersion struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Name          string
	Editor        string
	ContentFormat string
	Body          string
	Definition    map[string]any
	CreatedAt     time.Time
	ExportConfig  *TemplateExportConfig
}

type DocumentTemplateAssignment struct {
	DocumentID      string
	TemplateKey     string
	TemplateVersion int
	AssignedAt      time.Time
}

type DocumentTemplateSnapshot struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Editor        string
	ContentFormat string
	Body          string
	Definition    map[string]any
	ExportConfig  *TemplateExportConfig
}

func (v DocumentTemplateVersion) IsBrowserHTML() bool {
	return strings.EqualFold(v.Editor, "ckeditor5") && strings.EqualFold(v.ContentFormat, "html")
}

func (v DocumentTemplateVersion) IsMDDMEditor() bool {
	return strings.EqualFold(v.Editor, "mddm-blocknote") && strings.EqualFold(v.ContentFormat, "mddm")
}

func (v DocumentTemplateVersion) IsBrowserEditor() bool {
	return v.IsBrowserHTML() || v.IsMDDMEditor()
}

func (s DocumentTemplateSnapshot) IsBrowserHTML() bool {
	return strings.EqualFold(s.Editor, "ckeditor5") && strings.EqualFold(s.ContentFormat, "html")
}

func DefaultDocumentTemplateVersions() []DocumentTemplateVersion {
	return []DocumentTemplateVersion{
		{
			TemplateKey:   "po-default-canvas",
			Version:       1,
			ProfileCode:   "po",
			SchemaVersion: 3,
			Name:          "PO Governed Canvas v1",
			Editor:        "ckeditor5",
			ContentFormat: "html",
			Body: `<section class="md-doc-shell">
  <h1>Procedimento Operacional</h1>
  <p><strong>Objetivo</strong></p>
  <p><span class="restricted-editing-exception">Preencha o objetivo.</span></p>
  <p><strong>Descricao do processo</strong></p>
  <div class="restricted-editing-exception"><p>Descreva o processo.</p></div>
</section>`,
			Definition: map[string]any{
				"type": "page",
				"id":   "po-root",
				"children": []any{
					map[string]any{
						"type":  "section-frame",
						"id":    "identificacao-processo",
						"title": "Identificacao do Processo",
						"children": []any{
							map[string]any{"type": "label", "id": "lbl-objetivo", "text": "Objetivo"},
							map[string]any{"type": "field-slot", "id": "slot-objetivo", "path": "identificacaoProcesso.objetivo", "fieldKind": "scalar"},
							map[string]any{"type": "label", "id": "lbl-descricao", "text": "Descricao do processo"},
							map[string]any{"type": "rich-slot", "id": "slot-descricao", "path": "visaoGeral.descricaoProcesso", "fieldKind": "rich"},
						},
					},
				},
			},
			CreatedAt: time.Unix(0, 0).UTC(),
		},
		{
			TemplateKey:   "po-mddm-canvas",
			Version:       1,
			ProfileCode:   "po",
			SchemaVersion: 3,
			Name:          "PO MDDM Canvas v1",
			Editor:        "mddm-blocknote",
			ContentFormat: "mddm",
			Body:          "",
			Definition:    map[string]any{"type": "page", "id": "po-mddm-root", "children": []any{}},
			CreatedAt:     time.Unix(0, 0).UTC(),
		},
		{
			TemplateKey:   "po-default-browser",
			Version:       1,
			ProfileCode:   "po",
			SchemaVersion: 3,
			Name:          "Procedimento Operacional",
			Editor:        "ckeditor5",
			ContentFormat: "html",
			Body: `<section class="md-doc-shell">
  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr>
        <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">2. Identificação do Processo</td>
      </tr>
    </table>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Objetivo</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p><span class="restricted-editing-exception">Descreva o objetivo deste procedimento, incluindo o resultado esperado ao final da execução.</span></p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Escopo</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p><span class="restricted-editing-exception">Defina os limites de aplicação deste procedimento: onde começa, onde termina e o que está fora do escopo.</span></p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Cargo responsável</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p><span class="restricted-editing-exception">Informe o cargo ou função responsável pela execução deste procedimento.</span></p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Canal / Contexto</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p><span class="restricted-editing-exception">Descreva o canal ou contexto em que este procedimento se aplica.</span></p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Participantes</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p><span class="restricted-editing-exception">Liste os cargos, funções ou áreas que participam da execução deste procedimento.</span></p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr>
        <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">3. Entradas e Saídas</td>
      </tr>
    </table>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:22%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Entradas</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:28%;"><p><span class="restricted-editing-exception">Liste os insumos, informações ou materiais necessários para iniciar o processo.</span></p></td>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:22%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Saídas</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:28%;"><p><span class="restricted-editing-exception">Liste os produtos, resultados ou entregas gerados ao final do processo.</span></p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:22%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Documentos relacionados</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:28%;"><p><span class="restricted-editing-exception">Liste documentos, formulários ou registros utilizados ou gerados durante o processo.</span></p></td>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:22%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Sistemas utilizados</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:28%;"><p><span class="restricted-editing-exception">Liste os sistemas, ferramentas ou plataformas utilizadas na execução do processo.</span></p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr>
        <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">4. Visão Geral do Processo</td>
      </tr>
    </table>
    <div class="md-field">
      <p class="md-field-label"><strong>Descrição do processo</strong></p>
      <p><span class="restricted-editing-exception">Descreva o processo de forma detalhada, incluindo o contexto, fluxo geral e principais decisões envolvidas.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Ferramenta do fluxograma</strong></p>
      <p><span class="restricted-editing-exception">Informe a ferramenta utilizada para criar o fluxograma (ex: Bizagi, Visio, Miro).</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Link do fluxograma</strong></p>
      <p><span class="restricted-editing-exception">Cole o link de acesso ao fluxograma do processo.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Diagrama</strong></p>
      <p><span class="restricted-editing-exception">Insira ou descreva o diagrama do processo. Pode utilizar imagens ou representações textuais.</span></p>
    </div>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr>
        <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">5. Detalhamento das Etapas</td>
      </tr>
    </table>
    <p class="md-section-hint">Descreva cada etapa como uma seção livre. Duplique o bloco abaixo para adicionar mais etapas.</p>
    <div class="md-free-block">
      <h3><span class="restricted-editing-exception">Etapa 1 — [Nome da etapa]</span></h3>
      <p><span class="restricted-editing-exception">Descreva esta etapa livremente. Adicione parágrafos, listas, referências a outros documentos ou qualquer informação relevante para descrever o que acontece nesta etapa do processo.</span></p>
    </div>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr>
        <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">6. Controle e Exceções</td>
      </tr>
    </table>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Pontos de controle</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p><span class="restricted-editing-exception">Descreva os pontos de verificação, aprovação ou controle existentes no processo.</span></p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Exceções e desvios</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p><span class="restricted-editing-exception">Descreva situações excepcionais e como devem ser tratadas.</span></p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr>
        <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">7. Indicadores de Desempenho</td>
      </tr>
    </table>
    <div class="md-field">
      <p class="md-field-label"><strong>KPIs</strong></p>
      <figure class="table md-table restricted-editing-exception">
        <table>
          <thead>
            <tr><th>Indicador / KPI</th><th>Meta</th><th>Frequência</th></tr>
          </thead>
          <tbody>
            <tr><td>Ex: Taxa de retrabalho</td><td>Ex: &lt; 5%</td><td>Ex: Mensal</td></tr>
          </tbody>
        </table>
      </figure>
    </div>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr>
        <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">8. Documentos e Referências</td>
      </tr>
    </table>
    <figure class="table md-table restricted-editing-exception">
      <table>
        <thead>
          <tr><th>Código</th><th>Título / Descrição</th><th>Link / Localização</th></tr>
        </thead>
        <tbody>
          <tr><td>Ex: PO-001</td><td>Ex: Procedimento de compras</td><td>Ex: /docs/po-001</td></tr>
        </tbody>
      </table>
    </figure>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr>
        <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">9. Glossário</td>
      </tr>
    </table>
    <figure class="table md-table restricted-editing-exception">
      <table>
        <thead>
          <tr><th>Termo</th><th>Definição</th></tr>
        </thead>
        <tbody>
          <tr><td>Ex: SLA</td><td>Ex: Acordo de nível de serviço</td></tr>
        </tbody>
      </table>
    </figure>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr>
        <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">10. Histórico de Revisões</td>
      </tr>
    </table>
    <figure class="table md-table">
      <table>
        <thead>
          <tr><th>Versão</th><th>Data</th><th>O que foi alterado</th><th>Por</th></tr>
        </thead>
        <tbody>
          <tr>
            <td><p class="restricted-editing-exception">{{versao}}</p></td>
            <td><p class="restricted-editing-exception">{{data_criacao}}</p></td>
            <td><p class="restricted-editing-exception"></p></td>
            <td><p class="restricted-editing-exception">{{elaborador}}</p></td>
          </tr>
        </tbody>
      </table>
    </figure>
  </section>
</section>`,
			Definition: map[string]any{},
			CreatedAt:  time.Unix(0, 0).UTC(),
			ExportConfig: &TemplateExportConfig{
				MarginTop:    0.625,
				MarginRight:  0.625,
				MarginBottom: 0.625,
				MarginLeft:   0.625,
			},
		},
	}
}

package domain

import (
	"strings"
	"time"
)

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
}

func (v DocumentTemplateVersion) IsBrowserHTML() bool {
	return strings.EqualFold(v.Editor, "ckeditor5") && strings.EqualFold(v.ContentFormat, "html")
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
			TemplateKey:   "po-default-browser",
			Version:       1,
			ProfileCode:   "po",
			SchemaVersion: 3,
			Name:          "Procedimento Operacional",
			Editor:        "ckeditor5",
			ContentFormat: "html",
			Body: `<section class="md-doc-shell">
  <section class="md-section">
    <h2>2. Identificação do Processo</h2>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Objetivo</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva o objetivo deste procedimento, incluindo o resultado esperado ao final da execução.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Escopo</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Defina os limites de aplicação deste procedimento: onde começa, onde termina e o que está fora do escopo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Cargo responsável</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Informe o cargo ou função responsável pela execução deste procedimento.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Canal / Contexto</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva o canal ou contexto em que este procedimento se aplica.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Participantes</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os cargos, funções ou áreas que participam da execução deste procedimento.</p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <h2>3. Entradas e Saídas</h2>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Entradas</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os insumos, informações ou materiais necessários para iniciar o processo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Saídas</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os produtos, resultados ou entregas gerados ao final do processo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Documentos relacionados</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste documentos, formulários ou registros utilizados ou gerados durante o processo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Sistemas utilizados</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os sistemas, ferramentas ou plataformas utilizadas na execução do processo.</p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <h2>4. Visão Geral do Processo</h2>
    <div class="md-field">
      <p class="md-field-label"><strong>Descrição do processo</strong></p>
      <div class="restricted-editing-exception"><p>Descreva o processo de forma detalhada, incluindo o contexto, fluxo geral e principais decisões envolvidas.</p></div>
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
      <div class="restricted-editing-exception"><p>Insira ou descreva o diagrama do processo. Pode utilizar imagens ou representações textuais.</p></div>
    </div>
  </section>

  <section class="md-section">
    <h2>5. Detalhamento das Etapas</h2>
    <p class="md-section-hint">Descreva cada etapa como uma seção livre. Duplique o bloco abaixo para adicionar mais etapas.</p>
    <div class="md-free-block restricted-editing-exception">
      <h3>Etapa 1 — [Nome da etapa]</h3>
      <p>Descreva esta etapa livremente. Adicione parágrafos, listas, referências a outros documentos ou qualquer informação relevante para descrever o que acontece nesta etapa do processo.</p>
    </div>
  </section>

  <section class="md-section">
    <h2>6. Controle e Exceções</h2>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Pontos de controle</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva os pontos de verificação, aprovação ou controle existentes no processo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Exceções e desvios</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva situações excepcionais e como devem ser tratadas.</p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <h2>7. Indicadores de Desempenho</h2>
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
    <h2>8. Documentos e Referências</h2>
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
    <h2>9. Glossário</h2>
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
    <h2>10. Histórico de Revisões</h2>
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
		},
	}
}

-- 0058_update_po_browser_template.sql
-- Updates po-default-browser v1 body_html to the redesigned template
-- using the global document design system primitives (md-section, md-field,
-- md-free-block, md-table). Section 5 (Etapas) is now free-form.
-- IMPORTANT: body_html must match 0057 and the Go seed in domain/template.go.
-- TestPOBrowserTemplateGoSQLParity validates Go vs 0057 only; it does NOT read this file.

UPDATE metaldocs.document_template_versions
SET body_html = $$<section class="md-doc-shell">
  <section class="md-section">
    <h2>2. Identificação do Processo</h2>
    <div class="md-field">
      <p class="md-field-label"><strong>Objetivo</strong></p>
      <p><span class="restricted-editing-exception">Descreva o objetivo deste procedimento, incluindo o resultado esperado ao final da execução.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Escopo</strong></p>
      <p><span class="restricted-editing-exception">Defina os limites de aplicação deste procedimento: onde começa, onde termina e o que está fora do escopo.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Cargo responsável</strong></p>
      <p><span class="restricted-editing-exception">Informe o cargo ou função responsável pela execução deste procedimento.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Canal / Contexto</strong></p>
      <p><span class="restricted-editing-exception">Descreva o canal ou contexto em que este procedimento se aplica.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Participantes</strong></p>
      <p><span class="restricted-editing-exception">Liste os cargos, funções ou áreas que participam da execução deste procedimento.</span></p>
    </div>
  </section>

  <section class="md-section">
    <h2>3. Entradas e Saídas</h2>
    <div class="md-field">
      <p class="md-field-label"><strong>Entradas</strong></p>
      <p><span class="restricted-editing-exception">Liste os insumos, informações ou materiais necessários para iniciar o processo.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Saídas</strong></p>
      <p><span class="restricted-editing-exception">Liste os produtos, resultados ou entregas gerados ao final do processo.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Documentos relacionados</strong></p>
      <p><span class="restricted-editing-exception">Liste documentos, formulários ou registros utilizados ou gerados durante o processo.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Sistemas utilizados</strong></p>
      <p><span class="restricted-editing-exception">Liste os sistemas, ferramentas ou plataformas utilizadas na execução do processo.</span></p>
    </div>
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
    <div class="md-field">
      <p class="md-field-label"><strong>Pontos de controle</strong></p>
      <p><span class="restricted-editing-exception">Descreva os pontos de verificação, aprovação ou controle existentes no processo.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Exceções e desvios</strong></p>
      <p><span class="restricted-editing-exception">Descreva situações excepcionais e como devem ser tratadas.</span></p>
    </div>
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
</section>$$
WHERE template_key = 'po-default-browser' AND version = 1;

ALTER TABLE metaldocs.document_template_versions
  ADD COLUMN IF NOT EXISTS editor TEXT NOT NULL DEFAULT 'ckeditor5',
  ADD COLUMN IF NOT EXISTS content_format TEXT NOT NULL DEFAULT 'html',
  ADD COLUMN IF NOT EXISTS body_html TEXT NOT NULL DEFAULT '';

UPDATE metaldocs.document_template_versions
SET
  editor = 'ckeditor5',
  content_format = 'html',
  body_html = $$
  <section class="md-doc-shell">
    <h1>Procedimento Operacional</h1>
    <p><strong>Objetivo</strong></p>
    <p><span class="restricted-editing-exception">Preencha o objetivo.</span></p>
    <p><strong>Descricao do processo</strong></p>
    <div class="restricted-editing-exception"><p>Descreva o processo.</p></div>
  </section>
  $$
WHERE template_key = 'po-default-canvas' AND version = 1;

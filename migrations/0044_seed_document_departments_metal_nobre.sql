INSERT INTO metaldocs.document_departments (code, name, description, is_active)
VALUES
  ('sgq', 'SGQ', 'Sistema de Gestao da Qualidade', TRUE),
  ('operacoes', 'Operacoes', 'Operacao e execucao do processo', TRUE),
  ('manutencao', 'Manutencao', 'Manutencao de equipamentos e infraestrutura', TRUE),
  ('compras', 'Compras', 'Compras e suprimentos', TRUE),
  ('logistica', 'Logistica', 'Logistica e expedicao', TRUE),
  ('financeiro', 'Financeiro', 'Financeiro e controladoria', TRUE),
  ('comercial', 'Comercial', 'Relacionamento com clientes e vendas', TRUE),
  ('rh', 'RH', 'Recursos humanos', TRUE),
  ('ti', 'TI', 'Tecnologia da informacao', TRUE)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_active = TRUE;

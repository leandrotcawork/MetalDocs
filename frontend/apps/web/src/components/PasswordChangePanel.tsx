type PasswordChangePanelProps = {
  newPassword: string;
  confirmPassword: string;
  onNewPasswordChange: (value: string) => void;
  onConfirmPasswordChange: (value: string) => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
};

export function PasswordChangePanel(props: PasswordChangePanelProps) {
  return (
    <section className="panel auth-panel">
      <div className="panel-heading"><p className="kicker">Seguranca</p><h2>Troca obrigatoria de senha</h2></div>
      <form data-testid="password-change-form" className="stack" onSubmit={props.onSubmit}>
        <p className="hint">No primeiro acesso, a sessao atual ja comprova a senha temporaria. Defina apenas a nova senha para concluir a ativacao.</p>
        <input data-testid="password-new" type="password" placeholder="Nova senha" value={props.newPassword} onChange={(event) => props.onNewPasswordChange(event.target.value)} required />
        <input data-testid="password-confirm" type="password" placeholder="Confirmar nova senha" value={props.confirmPassword} onChange={(event) => props.onConfirmPasswordChange(event.target.value)} required />
        <button data-testid="password-submit" type="submit">Atualizar senha</button>
      </form>
    </section>
  );
}

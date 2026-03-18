type AuthShellProps = {
  identifier: string;
  password: string;
  message: string;
  error: string;
  onIdentifierChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
};

export function AuthShell(props: AuthShellProps) {
  return (
    <div className="app-shell auth-shell">
      <section className="hero auth-hero">
        <div>
          <p className="eyebrow">MetalDocs Access</p>
          <h1>Login real com sessao segura e banco persistente em Docker.</h1>
          <p className="hero-copy">O fluxo oficial agora usa cookie HTTP-only e IAM backend-first. O header tecnico deixou de ser o caminho principal.</p>
        </div>
        <form className="hero-panel stack" onSubmit={props.onSubmit} data-testid="login-form">
          <label><span>Username ou e-mail</span><input data-testid="login-identifier" value={props.identifier} onChange={(event) => props.onIdentifierChange(event.target.value)} required /></label>
          <label><span>Senha</span><input data-testid="login-password" type="password" value={props.password} onChange={(event) => props.onPasswordChange(event.target.value)} required /></label>
          <button data-testid="login-submit" type="submit">Entrar</button>
          {props.message && <p className="hint">{props.message}</p>}
          {props.error && <p className="hint">{props.error}</p>}
        </form>
      </section>
    </div>
  );
}

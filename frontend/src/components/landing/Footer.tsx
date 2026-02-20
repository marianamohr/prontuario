export default function Footer() {
  return (
    <footer className="border-t border-border bg-background py-8">
      <div className="container flex flex-col items-center justify-between gap-4 sm:flex-row">
        <span className="text-sm text-muted-foreground">
          © {new Date().getFullYear()} Camihealth. Todos os direitos reservados.
        </span>
        <div className="flex gap-6 text-sm text-muted-foreground">
          <a href="#" className="hover:text-foreground transition-colors">
            Termos de uso
          </a>
          <a href="#" className="hover:text-foreground transition-colors">
            Privacidade
          </a>
          <a href="#" className="hover:text-foreground transition-colors">
            Contato
          </a>
        </div>
      </div>
    </footer>
  )
}

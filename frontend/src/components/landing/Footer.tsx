export default function Footer() {
  const rawPhone = String(import.meta.env.VITE_CONTACT_WHATSAPP || '')
  const phone = rawPhone.replace(/\D/g, '')
  const message = String(import.meta.env.VITE_CONTACT_WHATSAPP_MESSAGE || 'Olá! Vim pelo CamiHealth e gostaria de falar com você.')
  const href = phone ? `https://wa.me/${phone}?text=${encodeURIComponent(message)}` : 'https://wa.me/'
  console.log('href', href)
  return (
    <footer className="border-t border-border bg-background py-8">
      <div className="container flex flex-col items-center justify-between gap-4 sm:flex-row">
        <span className="text-sm text-muted-foreground">
          © {new Date().getFullYear()} CamiHealth. Todos os direitos reservados.
        </span>
        <div className="flex gap-6 text-sm text-muted-foreground">
          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            title={phone ? 'Falar no WhatsApp' : 'Configure VITE_CONTACT_WHATSAPP para apontar para seu WhatsApp'}
            className="hover:text-foreground transition-colors"
          >
            Contato
          </a>
        </div>
      </div>
    </footer>
  )
}

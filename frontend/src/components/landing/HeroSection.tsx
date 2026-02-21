import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'


const heroImage = '/hero-image.jpg'
export default function HeroSection() {
  return (
    <section className="relative overflow-hidden bg-background">
      <nav className="container flex items-center justify-between py-5">
        <span className="text-2xl font-bold text-primary tracking-tight">
          CamiHealth
        </span>
        <Link to="/login">
          <Button size="sm">Entrar</Button>
        </Link>
      </nav>

      <div className="container grid items-center gap-12 py-16 md:py-24 lg:grid-cols-2">
        <div className="space-y-6 animate-fade-in-up">
          <h1 className="text-4xl font-extrabold leading-tight tracking-tight md:text-5xl lg:text-6xl text-foreground">
            Seu consultório,{' '}
            <span className="text-primary">sempre com você.</span>
          </h1>
          <p className="max-w-lg text-lg text-muted-foreground leading-relaxed">
            Prontuário, agenda, contratos e atendimentos em um só lugar.
            Feito para profissionais de saúde que atendem por conta própria.
          </p>
          <div className="flex flex-wrap gap-3 pt-2">
            <Link to="/login">
              <Button size="lg" className="text-base px-8 py-6 font-semibold shadow-lg">
                Acessar o sistema
              </Button>
            </Link>
            <a href="#features">
              <Button variant="outline" size="lg" className="text-base px-8 py-6">
                Conheça os recursos
              </Button>
            </a>
          </div>
        </div>

        <div className="relative animate-fade-in-up" style={{ animationDelay: '0.2s' }}>
          <div className="overflow-hidden rounded-2xl shadow-2xl border border-border">
            <img
              src={heroImage}
              alt="Interface do CamiHealth — sistema para profissionais de saúde"
              className="w-full h-auto object-cover"
              loading="eager"
            />
          </div>
        </div>
      </div>
    </section>
  )
}

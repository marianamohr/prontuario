import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'

export default function CTASection() {
  return (
    <section className="py-20 bg-background">
      <div className="container text-center max-w-2xl mx-auto">
        <h2 className="text-3xl font-bold text-foreground mb-4">
          Pronto para organizar seu consultório?
        </h2>
        <p className="text-muted-foreground mb-8 text-lg">
          Comece agora mesmo. Leva poucos minutos para ter tudo no digital.
        </p>
        <Link to="/login">
          <Button size="lg" className="text-base px-10 py-6 font-semibold shadow-lg">
            Acessar o CamiHealth
          </Button>
        </Link>
      </div>
    </section>
  )
}

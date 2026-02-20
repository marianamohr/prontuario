import { FileText, Clock, ShieldAlert, Layers } from 'lucide-react'

const problems = [
  { icon: FileText, pain: 'Fichas de papel e prontuários soltos', solution: 'Tudo digital, organizado e acessível' },
  { icon: Layers, pain: 'Vários sistemas que não conversam', solution: 'Um único lugar para tudo' },
  { icon: Clock, pain: 'Tempo perdido com planilhas e controle manual', solution: 'Automatize agenda, lembretes e contratos' },
  { icon: ShieldAlert, pain: 'Preocupação com segurança dos dados', solution: 'Dados protegidos e em conformidade com a LGPD' },
]

export default function ProblemsSection() {
  return (
    <section className="bg-section-alt py-20">
      <div className="container">
        <h2 className="text-center text-3xl font-bold text-foreground mb-4">
          Chega de improvisar no consultório
        </h2>
        <p className="text-center text-muted-foreground mb-12 max-w-2xl mx-auto">
          Sabemos que a rotina é corrida. O Camihealth resolve os problemas mais
          comuns de quem atende sozinho.
        </p>
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {problems.map((item) => (
            <div
              key={item.pain}
              className="rounded-xl bg-card p-6 shadow-sm border border-border flex flex-col gap-4"
            >
              <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-feature-icon-bg">
                <item.icon className="h-5 w-5 text-feature-icon-fg" />
              </div>
              <p className="text-sm text-muted-foreground line-through">
                {item.pain}
              </p>
              <p className="font-medium text-foreground">{item.solution}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

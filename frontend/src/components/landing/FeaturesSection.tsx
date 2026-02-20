import {
  ClipboardList,
  CalendarDays,
  FileSignature,
  Bell,
  ShieldCheck,
  Palette,
} from 'lucide-react'

const features = [
  {
    icon: ClipboardList,
    title: 'Prontuário digital',
    description: 'Registre evoluções, anexos e histórico do paciente de forma segura e organizada.',
  },
  {
    icon: CalendarDays,
    title: 'Agenda e agendamentos',
    description: 'Gerencie horários, encaixes e disponibilidade em um calendário intuitivo.',
  },
  {
    icon: FileSignature,
    title: 'Contratos com assinatura eletrônica',
    description: 'Envie contratos e termos para assinatura digital, sem papel.',
  },
  {
    icon: Bell,
    title: 'Lembretes para o paciente',
    description: 'Envie lembretes automáticos por WhatsApp para reduzir faltas.',
  },
  {
    icon: ShieldCheck,
    title: 'Segurança e privacidade (LGPD)',
    description: 'Dados criptografados e conformidade com a legislação brasileira.',
  },
  {
    icon: Palette,
    title: 'Aparência personalizável',
    description: 'Ajuste cores, logo e nome do consultório para a identidade do seu espaço.',
  },
]

export default function FeaturesSection() {
  return (
    <section id="features" className="bg-section-alt py-20">
      <div className="container">
        <h2 className="text-center text-3xl font-bold text-foreground mb-4">
          Tudo que seu consultório precisa
        </h2>
        <p className="text-center text-muted-foreground mb-12 max-w-2xl mx-auto">
          Recursos pensados para simplificar a rotina do profissional de saúde.
        </p>
        <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((f) => (
            <div
              key={f.title}
              className="rounded-xl bg-card p-6 shadow-sm border border-border hover:shadow-md transition-shadow"
            >
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-feature-icon-bg mb-4">
                <f.icon className="h-6 w-6 text-feature-icon-fg" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">
                {f.title}
              </h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                {f.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

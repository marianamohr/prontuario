import { Stethoscope, Brain, Apple, Ear, HeartPulse } from 'lucide-react'

const audiences = [
  { icon: Brain, label: 'Psicólogos' },
  { icon: Apple, label: 'Nutricionistas' },
  { icon: Stethoscope, label: 'Médicos' },
  { icon: Ear, label: 'Fonoaudiólogos' },
  { icon: HeartPulse, label: 'Outros profissionais' },
]

export default function AudienceSection() {
  return (
    <section className="py-20 bg-background">
      <div className="container text-center">
        <h2 className="text-3xl font-bold text-foreground mb-4">
          Para quem é o CamiHealth?
        </h2>
        <p className="text-muted-foreground mb-10 max-w-xl mx-auto">
          Para profissionais liberais da saúde que atendem sozinhos ou em
          consultório próprio.
        </p>
        <div className="flex flex-wrap justify-center gap-4">
          {audiences.map((a) => (
            <div
              key={a.label}
              className="flex items-center gap-2 rounded-full bg-secondary px-5 py-3 text-secondary-foreground font-medium"
            >
              <a.icon className="h-4 w-4" />
              <span>{a.label}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

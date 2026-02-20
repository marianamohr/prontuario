import HeroSection from '@/components/landing/HeroSection'
import ProblemsSection from '@/components/landing/ProblemsSection'
import AudienceSection from '@/components/landing/AudienceSection'
import FeaturesSection from '@/components/landing/FeaturesSection'
import CTASection from '@/components/landing/CTASection'
import Footer from '@/components/landing/Footer'

export function Landing() {
  return (
    <main>
      <HeroSection />
      <ProblemsSection />
      <AudienceSection />
      <FeaturesSection />
      <CTASection />
      <Footer />
    </main>
  )
}

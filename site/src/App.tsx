import { AllFeatures } from './sections/AllFeatures'
import { Automations } from './sections/Automations'
import { Compare } from './sections/Compare'
import { Cost } from './sections/Cost'
import { Features } from './sections/Features'
import { Footer } from './sections/Footer'
import { Header } from './sections/Header'
import { Hero } from './sections/Hero'
import { HostRepute } from './sections/HostRepute'
import { Install } from './sections/Install'
import { Providers } from './sections/Providers'

/** reputecast.com: the Landing board (design/boards/Landing.dc.html). */
export function App() {
  return (
    <div className="site flex flex-col">
      <a href="#main" className="skip">
        Skip to content
      </a>
      <Header />
      <main id="main" tabIndex={-1}>
        <Hero />
        <Features />
        <Providers />
        <Automations />
        <AllFeatures />
        <Compare />
        <Cost />
        <Install />
        <HostRepute />
      </main>
      <Footer />
    </div>
  )
}

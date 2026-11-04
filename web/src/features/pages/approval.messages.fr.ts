import { translate } from '@/i18n'
import { m } from './approval.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  badge: 'Page',
  badgeLabel: 'Page\u00A0: ouvrir «\u00A0{name}\u00A0» dans l’éditeur de pages',
  line: 'Page · {path} · {visibility}',
  whenApproved: 'Après approbation',
  scheduled: '{date} à {time}',
  published: 'Publiée',
  cardTitle: 'Page',
  cardCaption: 'Ce que l’approbation publie\u00A0: le contenu, l’adresse, la visibilité et l’heure de publication',
  page: 'Page',
  pageValue: '{name} · {template}',
  address: 'Adresse',
  visibility: 'Visibilité',
  goesLive: 'Mise en ligne',
  goesLiveNow: 'Dès son approbation',
  goesLivePublished: 'Publiée dès son approbation',
  version: 'Version',
  versionValue: 'v{version}',
  previewTitle: 'Aperçu de «\u00A0{name}\u00A0», v{version}',
  previewFullTitle: 'Aperçu complet de «\u00A0{name}\u00A0»',
  previewCaption: 'La page telle que les lecteurs la verront, dans le cadre du site',
  previewUnavailable: 'L’aperçu n’est pas disponible\u00A0: {error}',
  previewLoading: 'Chargement de l’aperçu.',
  openPreview: 'Ouvrir l’aperçu complet',
  openBuilder: 'Ouvrir dans l’éditeur de pages',
  decisionNote: 'L’approbation porte sur cette version précise\u00A0: son contenu, son adresse, sa visibilité et son heure de publication. {when, select, at {Elle sera mise en ligne à son heure de publication} other {Elle sera mise en ligne dès son approbation}}. Si {name} la modifie avant, la publication nécessitera une nouvelle approbation.',
  rejectDetail: 'Une notification est envoyée à {name} et la page n’est pas publiée. Refusez quand elle ne doit pas du tout être mise en ligne\u202F; pour des corrections, demandez plutôt des modifications.',
  changesDetail: 'La demande revient à {name} avec votre commentaire. La page n’est pas publiée tant qu’une nouvelle demande n’est pas envoyée.',
})

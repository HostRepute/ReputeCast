import { translate } from '@/i18n'
import { m } from './approval.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  pushTitle: 'La notification',
  pushCaption: '{n, plural, one {# abonné} other {# abonnés}} sur {sites} · {schedule}',
  link: 'Lien',
  buttons: 'Boutons',
  expiry: 'Expire après',
  hours: '{n, plural, one {# heure} other {# heures}}',
  minutes: '{n, plural, one {# minute} other {# minutes}}',
  tag: 'Tag',
  none: '–',
  waTitle: 'Le message WhatsApp',
  waCaption: '{template} · {category} · {language} · {n, plural, one {# personne} other {# personnes}} · {schedule}',
  business: 'Envoyé en tant que',
  variables: 'Variables',
  varField: '\'{{\'{key}\'}}\' depuis {field}',
  varFieldFallback: '\'{{\'{key}\'}}\' depuis {field}, ou «\u00A0{fallback}\u00A0»',
  varText: '\'{{\'{key}\'}}\' {text}',
  cost: 'Coût estimé',
  costLine: '{amount}, facturé par Meta à la livraison',
})

import { translate } from '@/i18n'
import { mts } from './testsend.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(mts, {
  menuItem: 'Envoyer un e-mail de test…',
  title: 'Envoyer un test\u00A0: {key}',
  sendTo: 'Envoyer à',
  ownOnly: 'Sans transactional:send, les tests ne partent qu’à votre propre adresse.',
  fillWith: 'Remplir avec',
  samples: 'Les valeurs d’exemple',
  testData: 'Données de test · {name}',
  note: 'Le brouillon tel qu’il est maintenant, via la route Système, avec [Test] dans l’objet. Il n’est pas compté dans les chiffres du journal.',
  send: 'Envoyer un test',
  sending: 'Envoi…',
  sent: 'Test envoyé à {email}',
  sentWith: 'Test envoyé à {email} avec «\u00A0{name}\u00A0»',
})

import { translate } from '@/i18n'
import { m } from './senders.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  title: 'Dominio de enlaces cortos',
  sub: 'Cada enlace de un SMS se acorta aquí, con un código por destinatario. Los acortadores públicos como bit.ly están bloqueados: los operadores los filtran.',
  caption: 'Dominio de enlaces cortos',
  domain: 'Dominio',
  usedBy: 'Usado por',
  smsLinks: 'Enlaces en SMS',
  record: 'Registro',
  status: 'Estado',
  lastCheck: 'Última comprobación',
  cname: 'CNAME {name} → {target}',
  verifiedOn: 'Verificado el {date}',
  notVerified: 'Sin verificar',
  checkFailed: 'La comprobación falló',
  never: 'Aún sin comprobar',
  checkDns: 'Comprobar DNS',
  checked: 'DNS comprobado',
  manage: 'Enrutamiento y reglas',
  manageLabel: 'Cambiar el dominio de enlaces cortos en Enrutamiento y reglas',
  none: 'Aún no hay dominio de enlaces cortos. Los SMS con enlaces no se pueden enviar hasta que se verifique uno.',
  setUp: 'Configurar uno',
  untilVerified: 'Hasta que se verifique, no se pueden enviar SMS con enlaces. Los SMS sin enlaces, sí.',
})

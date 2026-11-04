import { translate } from '@/i18n'
import { m } from './messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  title: 'Administración de la instancia',
  documentTitle: '{page} · Administración de la instancia',
  adminsOnly: 'solo administradores',
  workspaceView: 'Vista del espacio de trabajo',
  workspaceSettings: 'Configuración del espacio de trabajo',
  tabsLabel: 'Secciones de administración de la instancia',
  notAdminBadge: '403 · solo administradores de la instancia',
  notAdminTitle: 'Solo quien administra la instancia puede abrir Administración de la instancia',
  backToOverview: 'Volver al resumen',
  account: 'Cuenta y seguridad',
  notAdminFootnote: 'Pide a quien administra la instancia los cambios en el inicio de sesión, el SSO o los espacios de trabajo.',
  notAdminBody: 'Administración de la instancia gestiona los usuarios, los métodos de inicio de sesión, el SSO, las políticas de seguridad y los espacios de trabajo de toda la instancia. Tus propios métodos de inicio de sesión están en Cuenta y seguridad.',
})

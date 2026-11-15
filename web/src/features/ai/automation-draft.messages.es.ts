import { translate } from '@/i18n'
import { ma } from './automation-draft.messages'

export default translate(ma, {
  title: 'Describir la automatización',
  lead: 'Redacta una secuencia de correos al unirse a una lista. Revisa los pasos y cada correo antes de guardar el conjunto.',
  brief: '¿Qué debe hacer esta secuencia?',
  hint: 'Describe los correos, las esperas y las ramas de interacción. No incluyas datos de contacto, credenciales ni URL.',
  name: 'Nombre de la automatización',
  list: 'Lista seleccionada',
  chooseList: 'Elige una lista',
  audience: 'Tras publicar manualmente: solo altas futuras confirmadas, una vez por persona y sin importaciones. Los lectores salen del flujo al dejar la lista. Guardar el borrador no incorpora a miembros actuales.',
  scope: 'Esta receta admite correo, espera, rama de interacción, división A/B y salida. Conserva los enlaces, imágenes y Liquid seleccionados. Otros disparadores, condiciones de contactos, webhooks y acciones de suscripción requieren el editor de flujos.',
  privacy: 'La descripción, el texto editorial seleccionado, los pasos permitidos y la voz guardada van al modelo. Los ID, nombres de listas y correos, destinos de enlaces y pies legales permanecen aquí. No se leen contactos ni se accede a URL.',
  waiting: 'Generando pasos y correos. No se ha guardado ningún flujo ni correo. Detener la espera puede generar cargos del proveedor.',
  saved: 'Automatización y correos guardados como borradores. Nada publicado y nadie incorporado.',
  rejected: 'Propuesta rechazada. No se ha cambiado ningún flujo ni correo.',
  leaveBody: 'Se cerrará la descripción sin guardar. Abre de nuevo este diálogo para revisar la propuesta antes de que caduque a las 24 horas. No se modifica ningún flujo ni correo.',
  unsaved: 'Espera a que se guarde el flujo antes de generar o aceptar un reemplazo.',
  replace: 'Compara el grafo y los correos guardados con la propuesta. Al aceptar se crean nuevas copias de los correos y se sustituye este grafo de borrador; los correos existentes no cambian.',
  unsupported: 'Esta receta requiere altas futuras confirmadas, una vez por persona, sin importaciones y con salida al dejar la lista. Conserva otras reglas de audiencia en el editor de flujos.',
  graph: 'Pasos y rutas',
  emails: 'Borradores de correo',
  exact: 'Revisar el grafo exacto',
  next: 'Siguiente: {steps}',
  end: 'Fin de la ruta',
  checked: 'Grafo y contenido de correo validados. Aún se aplican las comprobaciones de publicación.',
})

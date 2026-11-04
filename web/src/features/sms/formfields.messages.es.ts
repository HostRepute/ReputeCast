import { translate } from '@/i18n'
import { m } from './formfields.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  phone: 'Número de móvil',
  phoneOptional: 'Número de móvil · opcional',
  phoneRequired: 'Número de móvil · obligatorio',
  textsTitle: 'SMS',
  smsConsent: 'Casilla de consentimiento para SMS · sin marcar por defecto',
  needsPhone: 'Requiere el campo Número de móvil.',
  consentText: 'Texto de consentimiento para SMS',
  evidence: 'El número, las listas, este texto exacto y su versión, la página, la dirección IP y la hora se guardan como evidencia del consentimiento para SMS.',
  newVersion: 'Al guardar empieza la versión {version}; las suscripciones anteriores conservan el texto que aceptaron.',
  separate: 'El consentimiento para el correo nunca suscribe a nadie a los SMS: esta casilla es independiente.',
  confirmation: 'Cuando alguien la marca, recibe el SMS de confirmación de alta definido en Enrutamiento y reglas.',
  termsWords: 'condiciones de los SMS',
})

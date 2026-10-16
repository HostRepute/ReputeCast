'use strict';

// Input fields several steps share.

/**
 * Zapier gives apps a Zap's ID but not its name; ReputeCast lists "Zaps
 * using ReputeCast" by name, so the name is typed here once per step.
 */
const zapNameField = {
  key: 'zap_name',
  label: 'Zap Name',
  type: 'string',
  required: false,
  helpText:
    "Optional. ReputeCast lists the Zaps that use it by name (**Settings › Integrations › Zapier**), but Zapier doesn't tell apps a Zap's name. Type it here to tell your Zaps apart; without it, this Zap is listed as Unnamed Zap.",
};

const optInField = {
  key: 'consent',
  label: 'Opt-In',
  type: 'string',
  required: false,
  choices: {
    single: 'Single opt-in: add them to the list now',
    double: 'Double opt-in: send the confirmation email first',
  },
  helpText: "Leave it empty to use each list's own setting. ReputeCast never adds someone back to a list they left, and never adds a suppressed address.",
};

const emailField = (helpText) => ({
  key: 'email',
  label: 'Email',
  type: 'string',
  required: true,
  helpText,
});

module.exports = { zapNameField, optInField, emailField };

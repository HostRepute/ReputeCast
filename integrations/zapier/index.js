'use strict';

// ReputeCast for Zapier. It uses only ReputeCast's public API
// (docs/integrations/hub.md): anyone can build another integration the
// same way.

const authentication = require('./authentication');
const { addHeaders, handleErrors } = require('./lib/client');
const hooks = require('./triggers/hooks');
const dropdowns = require('./triggers/dropdowns');
const upsertContact = require('./creates/upsert_contact');
const addToList = require('./creates/add_to_list');
const addTag = require('./creates/add_tag');
const recordEvent = require('./creates/record_event');
const sendTransactional = require('./creates/send_transactional');

const byKey = (list) => Object.fromEntries(list.map((x) => [x.key, x]));

module.exports = {
  version: require('./package.json').version,
  platformVersion: require('zapier-platform-core').version,

  authentication,
  beforeRequest: [addHeaders],
  afterResponse: [handleErrors],

  triggers: byKey([...hooks, ...dropdowns]),
  creates: byKey([upsertContact, addToList, addTag, recordEvent, sendTransactional]),
};

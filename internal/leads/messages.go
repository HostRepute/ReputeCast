package leads

import "github.com/hostrepute/reputecast/internal/i18n"

// Problems and notifications a member reads follow their language
// (messages/<locale>.json). Audit entries and consent notes keep their
// English, with the message beside it.
var (
	mConnectionNotFound = msgs.M("connection.notFound")
	mConnectionGone     = msgs.M("connection.gone")
	mConnectionExists   = msgs.M("connection.exists")
	mNameMissing        = msgs.M("name.missing")
	mTokenRejected      = msgs.M("token.rejected")
	mTokenExpired       = msgs.M("token.expired")
	mTokenPermissions   = msgs.M("token.permissions")
	mNoPages            = msgs.M("token.noPages")
	mAppRejected        = msgs.M("app.rejected")
	mAppSecretWithID    = msgs.M("app.secretWithId")
	mMetaUnavailable    = msgs.M("meta.unavailable")
	mMetaRateLimited    = msgs.M("meta.rateLimited")
	mPageUnknown        = msgs.M("page.unknown")
	mSubscribeFailed    = msgs.M("page.subscribeFailed")
	mPageNotChosen      = msgs.M("page.notChosen")
	mFormNotFound       = msgs.M("form.notFound")
	mFormNotMapped      = msgs.M("form.notMapped")
	mImportPaused       = msgs.M("import.paused")
	mImportRunning      = msgs.M("import.running")
	mMappingQuestion    = msgs.M("mapping.question")
	mMappingTarget      = msgs.M("mapping.target")
	mMappingTwice       = msgs.M("mapping.twice")
	mMappingEmail       = msgs.M("mapping.email")
	mMappingNames       = msgs.M("mapping.names")
	mMappingList        = msgs.M("mapping.list")
	mMappingListScope   = msgs.M("mapping.listScope")
	mConsentNoCheckbox  = msgs.M("consent.noCheckbox")
	mConsentPick        = msgs.M("consent.pick")
	mNotifyTitle        = msgs.M("notify.title")
	mNotifyBody         = msgs.M("notify.body")
	mNoteLead           = msgs.M("note.lead")
)

// note is a consent note: its English, and the message for readers in
// other languages.
type note struct {
	English string
	Text    i18n.Text
}

// keptNote is a lead's consent note ("Facebook/Instagram lead form Spring
// webinar signup").
func keptNote(form string) note {
	t := mNoteLead.Keep(i18n.Args{"form": form})
	return note{English: t.In(i18n.Default), Text: t}
}

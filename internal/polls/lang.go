package polls

import (
	"embed"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// What members read about polls and surveys follows their app language
// (i18n.LangFrom): the problems that keep a poll or a survey from working,
// the report's breakdown labels, and the default name and thank-you text of
// a new survey. They are in messages/<locale>.json.
//
// What a contact reads stays in English for now: the note under a Poll
// block in the email and its faces' words (the composer writes the same
// MJML), and the survey and answer pages' own words (internal/publicpages),
// which have no language of their own yet.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "polls")

var (
	mPollQuestion   = msgs.M("problem.pollQuestion")
	mPollTwoAnswers = msgs.M("problem.pollTwoAnswers")
	mPollFive       = msgs.M("problem.pollFiveAnswers")
	mPollTags       = msgs.M("problem.pollTags")
	mPollField      = msgs.M("problem.pollField")
	mPollSurvey     = msgs.M("problem.pollSurvey")
	mPollClosed     = msgs.M("problem.pollClosed")
	mPollOpen       = msgs.M("problem.pollOpen")

	mSurveyFirst       = msgs.M("problem.surveyFirstQuestion")
	mSurveyMax         = msgs.M("problem.surveyMaxQuestions")
	mSurveyQuestion    = msgs.M("problem.surveyQuestion")
	mSurveyTwoAnswers  = msgs.M("problem.surveyTwoAnswers")
	mSurveyField       = msgs.M("problem.surveyField")
	mSurveyTags        = msgs.M("problem.surveyTags")
	mSurveySlugTaken   = msgs.M("problem.surveySlugTaken")
	mSurveyIncomplete  = msgs.M("problem.surveyIncomplete")
	mSurveyType        = msgs.M("problem.surveyType")
	mSurveyClosePassed = msgs.M("problem.surveyClosePassed")
	mSurveyNotOpen     = msgs.M("problem.surveyNotPublished")
	mSurveyCloseFuture = msgs.M("problem.surveyCloseFuture")

	mLabelUntitled = msgs.M("label.untitledSurvey")
	mLabelThanks   = msgs.M("label.surveyThanks")
	mLabelNoTier   = msgs.M("label.noTier")
	mLabelNotSet   = msgs.M("label.languageNotSet")
)

// tierLabel is an engagement tier's name in a breakdown: "At risk".
func tierLabel(l i18n.Lang, tier string) string {
	if m, err := msgs.Lookup("tier." + tier); err == nil {
		return m.Text(l, nil)
	}
	return audience.TierName(tier)
}

// languageLabel is a contact language's name in the reader's language:
// "Spanish", "Tiếng Tây Ban Nha" (CLDR's names, as the browser writes them).
func languageLabel(l i18n.Lang, code string) string {
	if l.Locale == i18n.Vietnamese {
		if n := display.Vietnamese.Languages().Name(language.Make(accessibility.BaseLanguage(code))); n != "" {
			return n
		}
	}
	return accessibility.LanguageName(code)
}

package segments

import (
	"embed"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
)

// What members read from segments follows their app language
// (i18n.LangFrom): the builder's fields, operators and values, condition
// and SQL problems, and contact source labels. The words are in
// messages/<locale>.json: field.<key>, op.<key> and option.<field>.<value>
// name the catalog by key, so a field another file adds needs its message
// there too (TestCatalogWordsHaveMessages checks).

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "segments")

var (
	mFieldCustom = msgs.M("field.custom")

	mTimeout     = msgs.M("problem.timeout")
	mSQLDisabled = msgs.M("problem.sqlDisabled")
	mName        = msgs.M("problem.name")
	mTooManyGrps = msgs.M("problem.tooManyGroups")
	mMatch       = msgs.M("problem.match")
	mEmptyGroup  = msgs.M("problem.emptyGroup")
	mTooManyCond = msgs.M("problem.tooManyConditions")
	mUnknownFld  = msgs.M("problem.unknownField")
	mUnknownOp   = msgs.M("problem.unknownOp")
	mPeriodRange = msgs.M("problem.periodRange")
	mPeriodUnit  = msgs.M("problem.periodUnit")
	mPickList    = msgs.M("problem.pickList")
	mListGone    = msgs.M("problem.listGone")
	mPickCamp    = msgs.M("problem.pickCampaign")
	mPickLang    = msgs.M("problem.pickLanguage")
	mPickValue   = msgs.M("problem.pickValue")
	mNumber      = msgs.M("problem.number")
	mDate        = msgs.M("problem.date")
	mEnterValue  = msgs.M("problem.enterValue")
	mBothEnds    = msgs.M("problem.bothEnds")
	mRangeSwap   = msgs.M("problem.rangeSwapNumbers")
	mRangeSwapAm = msgs.M("problem.rangeSwapAmounts")
	mPickTier    = msgs.M("problem.pickTier")
	mLTV         = msgs.M("problem.ltv")
	mScore       = msgs.M("problem.score")
	mScoreEnds   = msgs.M("problem.scoreBothEnds")
	mPickGrowth  = msgs.M("problem.pickGrowthSource")
	mPickStore   = msgs.M("problem.pickStore")
	mOrders      = msgs.M("problem.orders")
	mSpent       = msgs.M("problem.spent")
	mLastOrderR  = msgs.M("problem.lastOrderRange")
	mLastOrderU  = msgs.M("problem.lastOrderUnit")
	mPickBought  = msgs.M("problem.pickProductOrCollection")
	mPickProduct = msgs.M("problem.pickProduct")
	mPickPush    = msgs.M("problem.pickPushSite")
	mPickListAny = msgs.M("problem.pickListAny")
	mDaysAgo     = msgs.M("problem.daysAgo")
	mSignedIn    = msgs.M("problem.signedInDate")
	mReferrals   = msgs.M("problem.referrals")
	mPickMile    = msgs.M("problem.pickMilestone")
	mPickJoined  = msgs.M("problem.pickReferralOrPartner")

	mSQLEmpty      = msgs.M("sql.empty")
	mSQLTooLong    = msgs.M("sql.tooLong")
	mSQLOne        = msgs.M("sql.oneStatement")
	mSQLSelect     = msgs.M("sql.selectOnly")
	mSQLComment    = msgs.M("sql.comment")
	mSQLString     = msgs.M("sql.string")
	mSQLIdent      = msgs.M("sql.identifier")
	mSQLDollar     = msgs.M("sql.dollar")
	mSQLViews      = msgs.M("sql.views")
	mSQLIDUUID     = msgs.M("sql.idColumnUUID")
	mSQLID         = msgs.M("sql.idColumn")
	mSQLReadOnly   = msgs.M("sql.readOnly")
	mSQLRoles      = msgs.M("sql.roles")
	mNoLanguage    = msgs.M("option.language.none")
	mTierAndAbove  = msgs.M("option.paid_tier.andAbove")
	mAnyPushSite   = msgs.M("option.push_site.any")
	mAnyList       = msgs.M("option.whatsapp_consent.any")
	mAnyProduct    = msgs.M("option.bought_product.any")
	mAnyStore      = msgs.M("option.store.any")
	mCollection    = msgs.M("option.store_bought.collection")
	mStoreProduct  = msgs.M("option.store_bought.product")
	mJoinReferral  = msgs.M("option.joined_through.referral")
	mJoinPartner   = msgs.M("option.joined_through.partner")
	mJoinPartnerOf = msgs.M("option.joined_through.partnerNamed")
	mMilestone     = msgs.M("option.referral_milestone.milestone")

	mSourceImport  = msgs.M("source.import")
	mSourcePopup   = msgs.M("source.popup")
	mSourceForm    = msgs.M("source.form")
	mSourceFormOf  = msgs.M("source.formNamed")
	mSourceAPI     = msgs.M("source.api")
	mSourceManual  = msgs.M("source.manual")
	mSourceProduct = msgs.M("source.productCheckout")
	mSourceRef     = msgs.M("source.referral")
	mSourcePartner = msgs.M("source.partner")
	mSourceUnknown = msgs.M("source.unknown")
	mImporterFile  = msgs.M("source.importerFile")
)

// fieldLabel is a field's name in a reader's language: "Mailbox provider",
// "Custom field · plan".
func fieldLabel(l i18n.Lang, key string) string {
	if k, ok := strings.CutPrefix(key, "custom:"); ok {
		return mFieldCustom.Text(l, i18n.Args{"key": k})
	}
	if m, err := msgs.Lookup("field." + key); err == nil {
		return m.Text(l, nil)
	}
	return key
}

// opLabel is an operator in a reader's language: "is not".
func opLabel(l i18n.Lang, key string) string {
	if m, err := msgs.Lookup("op." + key); err == nil {
		return m.Text(l, nil)
	}
	return key
}

// optionLabel is a fixed value of a field in a reader's language: "At
// risk". Names (Gmail, Chrome) have no message: they read as name.
func optionLabel(l i18n.Lang, field, value, name string) string {
	if m, err := msgs.Lookup("option." + field + "." + value); err == nil {
		return m.Text(l, nil)
	}
	if name == "" {
		return value
	}
	return name
}

// languageName is a content language's name in a reader's language:
// "Spanish", "Tiếng Tây Ban Nha" (as Intl.DisplayNames names it on the web).
func languageName(l i18n.Lang, code string) string {
	english := languages.NameOf(code)
	if l.Locale == "" || l.Locale == i18n.English || l.Locale == i18n.Pseudo {
		return english
	}
	if _, ok := languages.Lookup(code); !ok {
		return english
	}
	tag, err := language.Parse(languages.Base(code))
	if err != nil {
		return english
	}
	if n := display.Languages(l.Locale.Tag()).Name(tag); n != "" {
		return n
	}
	return english
}

// problem is a sentence saying what's wrong, written in each reader's
// language. When field is set, that field's name is the {field} argument
// (lower-cased inside a sentence: "Pick a list for list membership.").
type problem struct {
	msg   i18n.Msg
	args  i18n.Args
	field string
	lower bool
}

func (p *problem) text(l i18n.Lang) string {
	args := i18n.Args{}
	for k, v := range p.args {
		args[k] = v
	}
	if p.field != "" {
		label := fieldLabel(l, p.field)
		if p.lower {
			label = strings.ToLower(label)
		}
		args["field"] = label
	}
	return p.msg.Text(l, args)
}

func (p *problem) Error() string { return p.text(i18n.Default) }

// says returns a problem without arguments.
func says(m i18n.Msg) error { return &problem{msg: m} }

// sayf returns a problem with arguments.
func sayf(m i18n.Msg, args i18n.Args) error { return &problem{msg: m, args: args} }

// sayField returns a problem naming a condition's field.
func sayField(m i18n.Msg, field string, lower bool, args i18n.Args) error {
	return &problem{msg: m, args: args, field: field, lower: lower}
}

// textOf writes an error of this package in a reader's language: its
// message when it has one, else its English.
func textOf(l i18n.Lang, err error) string {
	if p, ok := err.(interface{ text(i18n.Lang) string }); ok {
		return p.text(l)
	}
	return err.Error()
}

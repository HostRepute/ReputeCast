package welcomeprompt

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The prompts' problems follow the reader's app language (i18n.LangFrom).
// They are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "welcomeprompt")

var (
	mNameFormOrImport = msgs.M("problem.nameFormOrImport")
	mNameList         = msgs.M("problem.nameList")
	mContext          = msgs.M("problem.context")
)

package api

import (
	"reflect"
	"strings"
	"testing"
)

type CampaignPath struct {
	CampaignID string `path:"campaignId"`
}

type campaignPath struct {
	CampaignID string `path:"campaignId"`
}

// aliasPath is how the reschedule input lost its campaign id in M45.
type aliasPath = campaignPath

type goodInput struct {
	WorkspaceInput
	CampaignPath
	Page int `query:"page"`
	Body struct {
		Name string `json:"name"`
	}
}

type unexportedEmbed struct {
	WorkspaceInput
	campaignPath
}

type aliasEmbed struct {
	WorkspaceInput
	aliasPath
}

type unexportedField struct {
	WorkspaceInput
	page int `query:"page"` //nolint:unused // the field under test
}

// Review 2026-11 CA-11: a parameter Huma can't set fails registration.
func TestParametersReachTheHandler(t *testing.T) {
	checkInput(reflect.TypeFor[goodInput](), "good")
	for name, typ := range map[string]reflect.Type{
		"unexported embed": reflect.TypeFor[unexportedEmbed](),
		"aliased embed":    reflect.TypeFor[aliasEmbed](),
		"unexported field": reflect.TypeFor[unexportedField](),
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil || !strings.Contains(r.(string), "never receives it") {
					t.Fatalf("no panic for %s: %v", name, r)
				}
			}()
			checkInput(typ, "bad")
		})
	}
}

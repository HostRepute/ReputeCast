from enum import StrEnum

class AIModelViewFeaturesItem(StrEnum):
    ACCESSIBILITY_FIX = "accessibility_fix"
    AGENT = "agent"
    ALT_TEXT = "alt_text"
    ASSISTANT = "assistant"
    BRAND_VOICE = "brand_voice"
    DRAFT = "draft"
    REPORT_SUMMARY = "report_summary"
    REWRITE = "rewrite"
    SEGMENT_DRAFT = "segment_draft"
    SUBJECT_IDEAS = "subject_ideas"
    TRANSLATE = "translate"

    def __str__(self) -> str:
        return str(self.value)

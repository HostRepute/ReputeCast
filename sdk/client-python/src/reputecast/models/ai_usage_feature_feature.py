from enum import StrEnum

class AIUsageFeatureFeature(StrEnum):
    ACCESSIBILITY_FIX = "accessibility_fix"
    ALT_TEXT = "alt_text"
    REPORT_SUMMARY = "report_summary"
    REWRITE = "rewrite"
    SEGMENT_DRAFT = "segment_draft"
    SUBJECT_IDEAS = "subject_ideas"
    TRANSLATE = "translate"

    def __str__(self) -> str:
        return str(self.value)

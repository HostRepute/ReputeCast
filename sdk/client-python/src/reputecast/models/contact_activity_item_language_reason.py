from enum import StrEnum

class ContactActivityItemLanguageReason(StrEnum):
    DRAFT = "draft"
    HELD = "held"
    MATCH = "match"
    MISSING = "missing"
    NOT_OFFERED = "not_offered"
    NO_LANGUAGE = "no_language"
    REQUESTED = "requested"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)

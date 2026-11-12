from enum import StrEnum

class DigestIssueSkipReason(StrEnum):
    FEED = "feed"
    HELD = "held"
    NOT_APPROVED = "not_approved"
    NO_ITEMS = "no_items"
    PERSON = "person"
    VALUE_5 = ""

    def __str__(self) -> str:
        return str(self.value)

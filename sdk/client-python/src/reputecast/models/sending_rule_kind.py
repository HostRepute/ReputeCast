from enum import StrEnum

class SendingRuleKind(StrEnum):
    CONTAINS_TEXT = "contains_text"
    LINK_DOMAIN = "link_domain"
    LINK_PARAM = "link_param"
    SUBJECT_LENGTH = "subject_length"

    def __str__(self) -> str:
        return str(self.value)

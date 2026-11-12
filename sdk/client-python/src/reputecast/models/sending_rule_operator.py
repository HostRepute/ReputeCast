from enum import StrEnum

class SendingRuleOperator(StrEnum):
    AT_LEAST = "at_least"
    AT_MOST = "at_most"
    BODY_CONTAINS = "body_contains"
    FOOTER_CONTAINS = "footer_contains"
    FOOTER_LACKS = "footer_lacks"
    NOT_ONE_OF = "not_one_of"
    ONE_OF = "one_of"
    ON_EVERY = "on_every"
    ON_NONE = "on_none"
    SUBJECT_CONTAINS = "subject_contains"

    def __str__(self) -> str:
        return str(self.value)

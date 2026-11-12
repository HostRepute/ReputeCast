from enum import StrEnum

class ScoringRuleKind(StrEnum):
    API_EVENT = "api_event"
    CLICK = "click"
    FIELD = "field"
    FORM = "form"
    OPEN = "open"
    SOFT_BOUNCE = "soft_bounce"
    UNSUBSCRIBE = "unsubscribe"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ContactScoreCauseKind(StrEnum):
    API_EVENT = "api_event"
    CLICK = "click"
    DECAY = "decay"
    FORM = "form"
    OPEN = "open"
    SOFT_BOUNCE = "soft_bounce"
    UNSUBSCRIBE = "unsubscribe"

    def __str__(self) -> str:
        return str(self.value)

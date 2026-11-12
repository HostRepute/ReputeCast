from enum import StrEnum

class LiveThresholdKey(StrEnum):
    API_ERRORS = "api_errors"
    BLOCK_BOUNCE = "block_bounce"
    BOUNCE = "bounce"
    COMPLAINT = "complaint"
    DEFERRAL = "deferral"
    MAILBOX_COMPLAINT = "mailbox_complaint"
    REPLY_RULES = "reply_rules"

    def __str__(self) -> str:
        return str(self.value)

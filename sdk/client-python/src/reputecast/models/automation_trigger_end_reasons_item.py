from enum import StrEnum

class AutomationTriggerEndReasonsItem(StrEnum):
    CANCELED = "canceled"
    PAYMENT_FAILED = "payment_failed"
    TRIAL_ENDED = "trial_ended"

    def __str__(self) -> str:
        return str(self.value)

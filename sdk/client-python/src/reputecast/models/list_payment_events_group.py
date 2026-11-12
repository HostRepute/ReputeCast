from enum import StrEnum

class ListPaymentEventsGroup(StrEnum):
    ALL = "all"
    CHANGES = "changes"
    FAILURES = "failures"
    PAYMENTS = "payments"
    REFUNDS = "refunds"
    TRIALS = "trials"

    def __str__(self) -> str:
        return str(self.value)

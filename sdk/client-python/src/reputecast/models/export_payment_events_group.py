from enum import StrEnum

class ExportPaymentEventsGroup(StrEnum):
    ALL = "all"
    CHANGES = "changes"
    FAILURES = "failures"
    PAYMENTS = "payments"
    REFUNDS = "refunds"
    TRIALS = "trials"

    def __str__(self) -> str:
        return str(self.value)

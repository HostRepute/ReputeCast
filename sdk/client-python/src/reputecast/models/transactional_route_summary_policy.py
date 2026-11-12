from enum import StrEnum

class TransactionalRouteSummaryPolicy(StrEnum):
    AUTOMATIC = "automatic"
    PAUSE_AND_ASK = "pause_and_ask"
    VALUE_3 = ""
    WARM_ONLY = "warm_only"

    def __str__(self) -> str:
        return str(self.value)

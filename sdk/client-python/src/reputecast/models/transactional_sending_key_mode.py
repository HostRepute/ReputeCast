from enum import StrEnum

class TransactionalSendingKeyMode(StrEnum):
    LIVE = "live"
    TEST = "test"

    def __str__(self) -> str:
        return str(self.value)

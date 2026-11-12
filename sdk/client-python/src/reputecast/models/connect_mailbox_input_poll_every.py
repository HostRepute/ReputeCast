from enum import StrEnum

class ConnectMailboxInputPollEvery(StrEnum):
    VALUE_0 = "1m"
    VALUE_1 = "5m"
    VALUE_2 = "15m"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class SendingDefaultsSuppressionSync(StrEnum):
    VALUE_0 = "15m"
    VALUE_1 = "1h"

    def __str__(self) -> str:
        return str(self.value)

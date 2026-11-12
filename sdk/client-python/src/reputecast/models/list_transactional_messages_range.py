from enum import StrEnum

class ListTransactionalMessagesRange(StrEnum):
    CUSTOM = "custom"
    VALUE_0 = "1h"
    VALUE_1 = "24h"
    VALUE_2 = "7d"
    VALUE_3 = "30d"

    def __str__(self) -> str:
        return str(self.value)

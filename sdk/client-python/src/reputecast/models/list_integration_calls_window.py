from enum import StrEnum

class ListIntegrationCallsWindow(StrEnum):
    VALUE_0 = "24h"
    VALUE_1 = "7d"

    def __str__(self) -> str:
        return str(self.value)

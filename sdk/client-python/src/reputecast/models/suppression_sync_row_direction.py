from enum import StrEnum

class SuppressionSyncRowDirection(StrEnum):
    IMPORT = "import"
    LOCAL = "local"
    OFF = "off"
    PULL = "pull"
    PUSH = "push"
    TWO_WAY = "two_way"

    def __str__(self) -> str:
        return str(self.value)

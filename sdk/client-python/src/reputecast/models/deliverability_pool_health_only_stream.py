from enum import StrEnum

class DeliverabilityPoolHealthOnlyStream(StrEnum):
    AUTOMATION = "automation"
    BROADCAST = "broadcast"
    SYSTEM = "system"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class SendingRuleAppliesTo(StrEnum):
    AUTOMATION = "automation"
    BOTH = "both"
    BROADCAST = "broadcast"

    def __str__(self) -> str:
        return str(self.value)

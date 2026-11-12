from enum import StrEnum

class PushEventSourceKind(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"
    CONFIRMATION = "confirmation"
    REPLY = "reply"
    TEST = "test"

    def __str__(self) -> str:
        return str(self.value)

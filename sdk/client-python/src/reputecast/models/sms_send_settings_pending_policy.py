from enum import StrEnum

class SmsSendSettingsPendingPolicy(StrEnum):
    SKIP = "skip"
    WAIT = "wait"

    def __str__(self) -> str:
        return str(self.value)

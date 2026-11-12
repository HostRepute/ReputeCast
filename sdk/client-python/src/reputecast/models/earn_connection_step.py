from enum import StrEnum

class EarnConnectionStep(StrEnum):
    CHECK = "check"
    DONE = "done"
    SETTINGS = "settings"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

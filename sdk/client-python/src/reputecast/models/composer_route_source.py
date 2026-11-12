from enum import StrEnum

class ComposerRouteSource(StrEnum):
    CAMPAIGN = "campaign"
    NONE = "none"
    RULE = "rule"

    def __str__(self) -> str:
        return str(self.value)

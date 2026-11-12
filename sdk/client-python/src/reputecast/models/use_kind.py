from enum import StrEnum

class UseKind(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"
    DIGEST = "digest"
    FORM = "form"
    SCORING = "scoring"
    SEGMENT = "segment"
    TRANSACTIONAL = "transactional"
    VIEW = "view"

    def __str__(self) -> str:
        return str(self.value)

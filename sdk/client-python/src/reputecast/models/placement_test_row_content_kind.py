from enum import StrEnum

class PlacementTestRowContentKind(StrEnum):
    CAMPAIGN = "campaign"
    DIGEST = "digest"
    TEMPLATE = "template"
    TRANSACTIONAL = "transactional"

    def __str__(self) -> str:
        return str(self.value)

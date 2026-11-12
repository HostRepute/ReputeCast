from enum import StrEnum

class RecommendationItemKind(StrEnum):
    LINK = "link"
    PARTNER = "partner"

    def __str__(self) -> str:
        return str(self.value)

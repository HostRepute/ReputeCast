from enum import StrEnum

class PushSiteKind(StrEnum):
    PUBLICATION = "publication"
    SNIPPET = "snippet"

    def __str__(self) -> str:
        return str(self.value)

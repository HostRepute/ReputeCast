from enum import StrEnum

class PushPromptSiteOptionKind(StrEnum):
    PUBLICATION = "publication"
    SNIPPET = "snippet"

    def __str__(self) -> str:
        return str(self.value)

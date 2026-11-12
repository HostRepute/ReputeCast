from enum import StrEnum

class EarnPageTierVisibility(StrEnum):
    ARCHIVED = "archived"
    HIDDEN = "hidden"
    PUBLIC = "public"

    def __str__(self) -> str:
        return str(self.value)

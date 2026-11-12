from enum import StrEnum

class PublicBrandingInputFont(StrEnum):
    CUSTOM = "custom"
    PLEX = "plex"
    SYSTEM = "system"

    def __str__(self) -> str:
        return str(self.value)

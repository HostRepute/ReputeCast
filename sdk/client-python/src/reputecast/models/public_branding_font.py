from enum import StrEnum

class PublicBrandingFont(StrEnum):
    CUSTOM = "custom"
    PLEX = "plex"
    SYSTEM = "system"

    def __str__(self) -> str:
        return str(self.value)

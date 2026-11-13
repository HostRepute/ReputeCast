from enum import StrEnum

class SponsorRequestFileKind(StrEnum):
    IMAGE = "image"
    LOGO = "logo"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class SponsorBlockLayout(StrEnum):
    IMAGE_TOP = "image-top"
    LOGO_LEFT = "logo-left"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class TemplateGalleryMode(StrEnum):
    DIGEST = "digest"
    REGULAR = "regular"

    def __str__(self) -> str:
        return str(self.value)

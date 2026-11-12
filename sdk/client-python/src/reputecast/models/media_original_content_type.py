from enum import StrEnum

class MediaOriginalContentType(StrEnum):
    IMAGEGIF = "image/gif"
    IMAGEJPEG = "image/jpeg"
    IMAGEPNG = "image/png"
    IMAGEWEBP = "image/webp"

    def __str__(self) -> str:
        return str(self.value)

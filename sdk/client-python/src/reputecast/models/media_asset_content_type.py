from enum import StrEnum

class MediaAssetContentType(StrEnum):
    IMAGEGIF = "image/gif"
    IMAGEJPEG = "image/jpeg"
    IMAGEPNG = "image/png"

    def __str__(self) -> str:
        return str(self.value)

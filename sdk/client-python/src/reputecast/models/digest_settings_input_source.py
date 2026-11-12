from enum import StrEnum

class DigestSettingsInputSource(StrEnum):
    URL = "url"
    WORDPRESS = "wordpress"

    def __str__(self) -> str:
        return str(self.value)

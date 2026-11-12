from enum import StrEnum

class PreviewPublicPagePage(StrEnum):
    ARCHIVE = "archive"
    CONFIRM = "confirm"
    PREFERENCES = "preferences"
    SUBSCRIBE = "subscribe"
    UNSUBSCRIBE = "unsubscribe"

    def __str__(self) -> str:
        return str(self.value)

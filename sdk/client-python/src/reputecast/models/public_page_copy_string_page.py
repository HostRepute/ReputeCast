from enum import StrEnum

class PublicPageCopyStringPage(StrEnum):
    CONFIRM = "confirm"
    LISTS = "lists"
    PREFERENCES = "preferences"
    PRIVACY = "privacy"
    SUBSCRIBE = "subscribe"
    UNSUBSCRIBE = "unsubscribe"

    def __str__(self) -> str:
        return str(self.value)

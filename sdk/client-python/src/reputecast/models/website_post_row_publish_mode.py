from enum import StrEnum

class WebsitePostRowPublishMode(StrEnum):
    AT = "at"
    NOW = "now"
    WITH_SEND = "with_send"

    def __str__(self) -> str:
        return str(self.value)

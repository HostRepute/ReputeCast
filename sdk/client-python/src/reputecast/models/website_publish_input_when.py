from enum import StrEnum

class WebsitePublishInputWhen(StrEnum):
    AT = "at"
    NOW = "now"

    def __str__(self) -> str:
        return str(self.value)

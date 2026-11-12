from enum import StrEnum

class WebsiteNavItemKind(StrEnum):
    ARCHIVE = "archive"
    HOME = "home"
    PAGE = "page"
    TAG = "tag"
    URL = "url"

    def __str__(self) -> str:
        return str(self.value)

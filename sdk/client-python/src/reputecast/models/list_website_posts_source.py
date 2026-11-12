from enum import StrEnum

class ListWebsitePostsSource(StrEnum):
    CAMPAIGN = "campaign"
    VALUE_2 = ""
    WEB = "web"

    def __str__(self) -> str:
        return str(self.value)

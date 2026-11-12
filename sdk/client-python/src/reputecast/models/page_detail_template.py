from enum import StrEnum

class PageDetailTemplate(StrEnum):
    ABOUT = "about"
    BLANK = "blank"
    LAUNCH = "launch"
    LEAD_MAGNET = "lead_magnet"
    SALES = "sales"
    WEBINAR = "webinar"

    def __str__(self) -> str:
        return str(self.value)

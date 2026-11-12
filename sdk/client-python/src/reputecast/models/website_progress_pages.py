from enum import StrEnum

class WebsiteProgressPages(StrEnum):
    DONE = "done"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)

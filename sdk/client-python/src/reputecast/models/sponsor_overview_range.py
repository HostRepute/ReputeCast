from enum import StrEnum

class SponsorOverviewRange(StrEnum):
    ALL = "all"
    AROUND = "around"
    RECENT = "recent"
    UPCOMING = "upcoming"

    def __str__(self) -> str:
        return str(self.value)

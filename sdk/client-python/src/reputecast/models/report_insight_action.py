from enum import StrEnum

class ReportInsightAction(StrEnum):
    CHECK_DELISTING = "check_delisting"
    OPEN_POSTMASTER = "open_postmaster"
    OPEN_SNDS = "open_snds"
    SHOW_LINK = "show_link"

    def __str__(self) -> str:
        return str(self.value)

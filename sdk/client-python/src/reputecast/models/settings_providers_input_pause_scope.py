from enum import StrEnum

class SettingsProvidersInputPauseScope(StrEnum):
    CAMPAIGN = "campaign"
    ROUTE = "route"

    def __str__(self) -> str:
        return str(self.value)

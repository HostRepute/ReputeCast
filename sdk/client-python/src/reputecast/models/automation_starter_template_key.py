from enum import StrEnum

class AutomationStarterTemplateKey(StrEnum):
    ABANDONED_CART = "abandoned-cart"
    ONBOARDING = "onboarding"
    RE_ENGAGEMENT = "re-engagement"
    WELCOME_SERIES = "welcome-series"

    def __str__(self) -> str:
        return str(self.value)

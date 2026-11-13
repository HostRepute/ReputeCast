from enum import StrEnum

class IntegrationGroupKey(StrEnum):
    AI = "ai"
    AUTOMATION = "automation"
    COMMERCE = "commerce"
    DELIVERABILITY = "deliverability"
    DEVELOPER = "developer"
    LEADS = "leads"
    PAYMENTS = "payments"
    VERIFICATION = "verification"
    WEBSITES = "websites"

    def __str__(self) -> str:
        return str(self.value)

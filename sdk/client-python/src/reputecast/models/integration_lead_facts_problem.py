from enum import StrEnum

class IntegrationLeadFactsProblem(StrEnum):
    APP_REJECTED = "app_rejected"
    PERMISSION_MISSING = "permission_missing"
    TOKEN_EXPIRED = "token_expired"
    TOKEN_REJECTED = "token_rejected"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)

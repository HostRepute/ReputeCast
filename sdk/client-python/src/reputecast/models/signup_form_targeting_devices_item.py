from enum import StrEnum

class SignupFormTargetingDevicesItem(StrEnum):
    DESKTOP = "desktop"
    MOBILE = "mobile"
    TABLET = "tablet"

    def __str__(self) -> str:
        return str(self.value)

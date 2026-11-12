from enum import StrEnum

class PushCutPlatform(StrEnum):
    CHROME_ANDROID = "chrome_android"
    CHROME_MAC = "chrome_mac"
    CHROME_WINDOWS = "chrome_windows"
    SAFARI_IOS = "safari_ios"
    SAFARI_MAC = "safari_mac"

    def __str__(self) -> str:
        return str(self.value)

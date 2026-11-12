from enum import StrEnum

class PushReportBrowserBrowser(StrEnum):
    CHROME = "chrome"
    CHROME_ANDROID = "chrome_android"
    EDGE = "edge"
    FIREFOX = "firefox"
    OTHER = "other"
    SAFARI = "safari"
    SAFARI_IOS = "safari_ios"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ReadinessFixAction(StrEnum):
    ACCESSIBILITY_COLOR = "accessibility_color"
    ACCESSIBILITY_DECORATIVE = "accessibility_decorative"
    ACCESSIBILITY_ONE_CLICK = "accessibility_one_click"
    EXCLUDE_RISKY = "exclude_risky"
    EXCLUDE_SUNSET = "exclude_sunset"
    HOLD_LANGUAGE = "hold_language"
    INSERT_FOOTER = "insert_footer"
    MARK_LANGUAGE_TRANSLATED = "mark_language_translated"
    MARK_LANGUAGE_UP_TO_DATE = "mark_language_up_to_date"
    MINIFY = "minify"
    POLL_CLOSE_DAYS = "poll_close_days"
    RELEASE_LANGUAGE = "release_language"
    REPLACE_LINK = "replace_link"
    ROUTE_ONLY = "route_only"
    RULE_ADD_PARAM = "rule_add_param"
    SPONSOR_REFRESH_BLOCK = "sponsor_refresh_block"
    SPONSOR_REMOVE_BLOCK = "sponsor_remove_block"
    USE_SUBJECT = "use_subject"
    VERIFY_ADDRESSES = "verify_addresses"
    VISIBILITY_HIDE_BLOCK = "visibility_hide_block"
    VISIBILITY_REMOVE_CONDITION = "visibility_remove_condition"
    VISIBILITY_SHOW_EVERYONE = "visibility_show_everyone"

    def __str__(self) -> str:
        return str(self.value)

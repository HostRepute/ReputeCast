from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_report_browser_browser import PushReportBrowserBrowser






T = TypeVar("T", bound="PushReportBrowser")



@_attrs_define
class PushReportBrowser:
    """ 
        Attributes:
            browser (PushReportBrowserBrowser):
            clicks (int):
            sent (int):
            shown (int):
            shown_pct (float):
     """

    browser: PushReportBrowserBrowser
    clicks: int
    sent: int
    shown: int
    shown_pct: float





    def to_dict(self) -> dict[str, Any]:
        browser = self.browser.value

        clicks = self.clicks

        sent = self.sent

        shown = self.shown

        shown_pct = self.shown_pct


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "browser": browser,
            "clicks": clicks,
            "sent": sent,
            "shown": shown,
            "shownPct": shown_pct,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        browser = PushReportBrowserBrowser(d.pop("browser"))




        clicks = d.pop("clicks")

        sent = d.pop("sent")

        shown = d.pop("shown")

        shown_pct = d.pop("shownPct")

        push_report_browser = cls(
            browser=browser,
            clicks=clicks,
            sent=sent,
            shown=shown,
            shown_pct=shown_pct,
        )

        return push_report_browser


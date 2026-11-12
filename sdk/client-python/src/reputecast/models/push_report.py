from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_report_status import PushReportStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.push_report_browser import PushReportBrowser
  from ..models.push_report_point import PushReportPoint
  from ..models.push_report_site import PushReportSite
  from ..models.push_report_skipped import PushReportSkipped





T = TypeVar("T", bound="PushReport")



@_attrs_define
class PushReport:
    """ 
        Attributes:
            by_browser (list[PushReportBrowser]):
            by_site (list[PushReportSite]):
            can_send (bool):
            clicks (int):
            clicks24 (list[PushReportPoint]):
            clicks_pct (float):
            expired (int):
            failed (int):
            held (int):
            id (str):
            name (str):
            planned (int):
            queued (int):
            sent (int):
            sent_by (str):
            shown (int):
            shown_pct (float):
            sites (list[str]):
            sites_not_in (list[str]):
            skipped (PushReportSkipped):
            status (PushReportStatus):
            tag (str):
            ttl_minutes (int):
            unsubscribed (int):
            unsubscribed_pct (float):
            sent_at (datetime.datetime | Unset):
     """

    by_browser: list[PushReportBrowser]
    by_site: list[PushReportSite]
    can_send: bool
    clicks: int
    clicks24: list[PushReportPoint]
    clicks_pct: float
    expired: int
    failed: int
    held: int
    id: str
    name: str
    planned: int
    queued: int
    sent: int
    sent_by: str
    shown: int
    shown_pct: float
    sites: list[str]
    sites_not_in: list[str]
    skipped: PushReportSkipped
    status: PushReportStatus
    tag: str
    ttl_minutes: int
    unsubscribed: int
    unsubscribed_pct: float
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_report_browser import PushReportBrowser # noqa: PLC0415
        from ..models.push_report_point import PushReportPoint # noqa: PLC0415
        from ..models.push_report_site import PushReportSite # noqa: PLC0415
        from ..models.push_report_skipped import PushReportSkipped # noqa: PLC0415
        by_browser = []
        for by_browser_item_data in self.by_browser:
            by_browser_item = by_browser_item_data.to_dict()
            by_browser.append(by_browser_item)



        by_site = []
        for by_site_item_data in self.by_site:
            by_site_item = by_site_item_data.to_dict()
            by_site.append(by_site_item)



        can_send = self.can_send

        clicks = self.clicks

        clicks24 = []
        for clicks24_item_data in self.clicks24:
            clicks24_item = clicks24_item_data.to_dict()
            clicks24.append(clicks24_item)



        clicks_pct = self.clicks_pct

        expired = self.expired

        failed = self.failed

        held = self.held

        id = self.id

        name = self.name

        planned = self.planned

        queued = self.queued

        sent = self.sent

        sent_by = self.sent_by

        shown = self.shown

        shown_pct = self.shown_pct

        sites = self.sites



        sites_not_in = self.sites_not_in



        skipped = self.skipped.to_dict()

        status = self.status.value

        tag = self.tag

        ttl_minutes = self.ttl_minutes

        unsubscribed = self.unsubscribed

        unsubscribed_pct = self.unsubscribed_pct

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byBrowser": by_browser,
            "bySite": by_site,
            "canSend": can_send,
            "clicks": clicks,
            "clicks24": clicks24,
            "clicksPct": clicks_pct,
            "expired": expired,
            "failed": failed,
            "held": held,
            "id": id,
            "name": name,
            "planned": planned,
            "queued": queued,
            "sent": sent,
            "sentBy": sent_by,
            "shown": shown,
            "shownPct": shown_pct,
            "sites": sites,
            "sitesNotIn": sites_not_in,
            "skipped": skipped,
            "status": status,
            "tag": tag,
            "ttlMinutes": ttl_minutes,
            "unsubscribed": unsubscribed,
            "unsubscribedPct": unsubscribed_pct,
        })
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_report_browser import PushReportBrowser # noqa: PLC0415
        from ..models.push_report_point import PushReportPoint # noqa: PLC0415
        from ..models.push_report_site import PushReportSite # noqa: PLC0415
        from ..models.push_report_skipped import PushReportSkipped # noqa: PLC0415
        d = dict(src_dict)
        by_browser = []
        _by_browser = d.pop("byBrowser")
        for by_browser_item_data in (_by_browser):
            by_browser_item = PushReportBrowser.from_dict(by_browser_item_data)



            by_browser.append(by_browser_item)


        by_site = []
        _by_site = d.pop("bySite")
        for by_site_item_data in (_by_site):
            by_site_item = PushReportSite.from_dict(by_site_item_data)



            by_site.append(by_site_item)


        can_send = d.pop("canSend")

        clicks = d.pop("clicks")

        clicks24 = []
        _clicks24 = d.pop("clicks24")
        for clicks24_item_data in (_clicks24):
            clicks24_item = PushReportPoint.from_dict(clicks24_item_data)



            clicks24.append(clicks24_item)


        clicks_pct = d.pop("clicksPct")

        expired = d.pop("expired")

        failed = d.pop("failed")

        held = d.pop("held")

        id = d.pop("id")

        name = d.pop("name")

        planned = d.pop("planned")

        queued = d.pop("queued")

        sent = d.pop("sent")

        sent_by = d.pop("sentBy")

        shown = d.pop("shown")

        shown_pct = d.pop("shownPct")

        sites = cast(list[str], d.pop("sites"))


        sites_not_in = cast(list[str], d.pop("sitesNotIn"))


        skipped = PushReportSkipped.from_dict(d.pop("skipped"))




        status = PushReportStatus(d.pop("status"))




        tag = d.pop("tag")

        ttl_minutes = d.pop("ttlMinutes")

        unsubscribed = d.pop("unsubscribed")

        unsubscribed_pct = d.pop("unsubscribedPct")

        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        push_report = cls(
            by_browser=by_browser,
            by_site=by_site,
            can_send=can_send,
            clicks=clicks,
            clicks24=clicks24,
            clicks_pct=clicks_pct,
            expired=expired,
            failed=failed,
            held=held,
            id=id,
            name=name,
            planned=planned,
            queued=queued,
            sent=sent,
            sent_by=sent_by,
            shown=shown,
            shown_pct=shown_pct,
            sites=sites,
            sites_not_in=sites_not_in,
            skipped=skipped,
            status=status,
            tag=tag,
            ttl_minutes=ttl_minutes,
            unsubscribed=unsubscribed,
            unsubscribed_pct=unsubscribed_pct,
            sent_at=sent_at,
        )

        return push_report


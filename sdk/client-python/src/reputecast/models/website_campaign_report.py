from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_campaign_report_status import WebsiteCampaignReportStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_source import WebsiteSource





T = TypeVar("T", bound="WebsiteCampaignReport")



@_attrs_define
class WebsiteCampaignReport:
    """ 
        Attributes:
            from_ (datetime.datetime):
            post_id (str):
            referrers (list[WebsiteSource]):
            signups (int):
            status (WebsiteCampaignReportStatus):
            title (str):
            until (datetime.datetime):
            upgrades (int):
            url (str):
            utm (list[WebsiteSource]):
            views (int):
            views_all_time (int):
            published_at (datetime.datetime | Unset):
     """

    from_: datetime.datetime
    post_id: str
    referrers: list[WebsiteSource]
    signups: int
    status: WebsiteCampaignReportStatus
    title: str
    until: datetime.datetime
    upgrades: int
    url: str
    utm: list[WebsiteSource]
    views: int
    views_all_time: int
    published_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_source import WebsiteSource # noqa: PLC0415
        from_ = self.from_.isoformat()

        post_id = self.post_id

        referrers = []
        for referrers_item_data in self.referrers:
            referrers_item = referrers_item_data.to_dict()
            referrers.append(referrers_item)



        signups = self.signups

        status = self.status.value

        title = self.title

        until = self.until.isoformat()

        upgrades = self.upgrades

        url = self.url

        utm = []
        for utm_item_data in self.utm:
            utm_item = utm_item_data.to_dict()
            utm.append(utm_item)



        views = self.views

        views_all_time = self.views_all_time

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from": from_,
            "postId": post_id,
            "referrers": referrers,
            "signups": signups,
            "status": status,
            "title": title,
            "until": until,
            "upgrades": upgrades,
            "url": url,
            "utm": utm,
            "views": views,
            "viewsAllTime": views_all_time,
        })
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_source import WebsiteSource # noqa: PLC0415
        d = dict(src_dict)
        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        post_id = d.pop("postId")

        referrers = []
        _referrers = d.pop("referrers")
        for referrers_item_data in (_referrers):
            referrers_item = WebsiteSource.from_dict(referrers_item_data)



            referrers.append(referrers_item)


        signups = d.pop("signups")

        status = WebsiteCampaignReportStatus(d.pop("status"))




        title = d.pop("title")

        until = datetime.datetime.fromisoformat(d.pop("until"))




        upgrades = d.pop("upgrades")

        url = d.pop("url")

        utm = []
        _utm = d.pop("utm")
        for utm_item_data in (_utm):
            utm_item = WebsiteSource.from_dict(utm_item_data)



            utm.append(utm_item)


        views = d.pop("views")

        views_all_time = d.pop("viewsAllTime")

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        website_campaign_report = cls(
            from_=from_,
            post_id=post_id,
            referrers=referrers,
            signups=signups,
            status=status,
            title=title,
            until=until,
            upgrades=upgrades,
            url=url,
            utm=utm,
            views=views,
            views_all_time=views_all_time,
            published_at=published_at,
        )

        return website_campaign_report


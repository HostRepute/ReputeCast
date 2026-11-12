from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_campaign_web_chip import WebsiteCampaignWebChip
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_post import WebsitePost





T = TypeVar("T", bound="WebsiteCampaignWeb")



@_attrs_define
class WebsiteCampaignWeb:
    """ 
        Attributes:
            audience_lists (list[str]):
            campaign_status (str):
            chip (WebsiteCampaignWebChip):
            follows_audience (bool):
            host (str):
            post (WebsitePost):
            site_on (bool):
            scheduled_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
     """

    audience_lists: list[str]
    campaign_status: str
    chip: WebsiteCampaignWebChip
    follows_audience: bool
    host: str
    post: WebsitePost
    site_on: bool
    scheduled_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_post import WebsitePost # noqa: PLC0415
        audience_lists = self.audience_lists



        campaign_status = self.campaign_status

        chip = self.chip.value

        follows_audience = self.follows_audience

        host = self.host

        post = self.post.to_dict()

        site_on = self.site_on

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audienceLists": audience_lists,
            "campaignStatus": campaign_status,
            "chip": chip,
            "followsAudience": follows_audience,
            "host": host,
            "post": post,
            "siteOn": site_on,
        })
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_post import WebsitePost # noqa: PLC0415
        d = dict(src_dict)
        audience_lists = cast(list[str], d.pop("audienceLists"))


        campaign_status = d.pop("campaignStatus")

        chip = WebsiteCampaignWebChip(d.pop("chip"))




        follows_audience = d.pop("followsAudience")

        host = d.pop("host")

        post = WebsitePost.from_dict(d.pop("post"))




        site_on = d.pop("siteOn")

        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        website_campaign_web = cls(
            audience_lists=audience_lists,
            campaign_status=campaign_status,
            chip=chip,
            follows_audience=follows_audience,
            host=host,
            post=post,
            site_on=site_on,
            scheduled_at=scheduled_at,
            sent_at=sent_at,
        )

        return website_campaign_web


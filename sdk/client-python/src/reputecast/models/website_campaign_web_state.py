from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_campaign_web_state_chip import WebsiteCampaignWebStateChip
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WebsiteCampaignWebState")



@_attrs_define
class WebsiteCampaignWebState:
    """ 
        Attributes:
            campaign_id (str):
            chip (WebsiteCampaignWebStateChip):
            featured (bool):
            post_id (str | Unset):
            publish_at (datetime.datetime | Unset):
            published_at (datetime.datetime | Unset):
            slug (str | Unset):
            unpublished_at (datetime.datetime | Unset):
            url (str | Unset):
            visibility (str | Unset):
     """

    campaign_id: str
    chip: WebsiteCampaignWebStateChip
    featured: bool
    post_id: str | Unset = UNSET
    publish_at: datetime.datetime | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET
    slug: str | Unset = UNSET
    unpublished_at: datetime.datetime | Unset = UNSET
    url: str | Unset = UNSET
    visibility: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        campaign_id = self.campaign_id

        chip = self.chip.value

        featured = self.featured

        post_id = self.post_id

        publish_at: str | Unset = UNSET
        if not isinstance(self.publish_at, Unset):
            publish_at = self.publish_at.isoformat()

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        slug = self.slug

        unpublished_at: str | Unset = UNSET
        if not isinstance(self.unpublished_at, Unset):
            unpublished_at = self.unpublished_at.isoformat()

        url = self.url

        visibility = self.visibility


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignId": campaign_id,
            "chip": chip,
            "featured": featured,
        })
        if post_id is not UNSET:
            field_dict["postId"] = post_id
        if publish_at is not UNSET:
            field_dict["publishAt"] = publish_at
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if slug is not UNSET:
            field_dict["slug"] = slug
        if unpublished_at is not UNSET:
            field_dict["unpublishedAt"] = unpublished_at
        if url is not UNSET:
            field_dict["url"] = url
        if visibility is not UNSET:
            field_dict["visibility"] = visibility

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaign_id = d.pop("campaignId")

        chip = WebsiteCampaignWebStateChip(d.pop("chip"))




        featured = d.pop("featured")

        post_id = d.pop("postId", UNSET)

        _publish_at = d.pop("publishAt", UNSET)
        publish_at: datetime.datetime | Unset
        if isinstance(_publish_at,  Unset):
            publish_at = UNSET
        else:
            publish_at = datetime.datetime.fromisoformat(_publish_at)




        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        slug = d.pop("slug", UNSET)

        _unpublished_at = d.pop("unpublishedAt", UNSET)
        unpublished_at: datetime.datetime | Unset
        if isinstance(_unpublished_at,  Unset):
            unpublished_at = UNSET
        else:
            unpublished_at = datetime.datetime.fromisoformat(_unpublished_at)




        url = d.pop("url", UNSET)

        visibility = d.pop("visibility", UNSET)

        website_campaign_web_state = cls(
            campaign_id=campaign_id,
            chip=chip,
            featured=featured,
            post_id=post_id,
            publish_at=publish_at,
            published_at=published_at,
            slug=slug,
            unpublished_at=unpublished_at,
            url=url,
            visibility=visibility,
        )

        return website_campaign_web_state


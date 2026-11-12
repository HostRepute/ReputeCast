from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationSiteDraft")



@_attrs_define
class IntegrationSiteDraft:
    """ 
        Attributes:
            created_at (datetime.datetime):
            status (str): The campaign's status (draft, scheduled, sent…)
            title (str):
            audience (str | Unset):
            campaign_id (str | Unset):
            notified (str | Unset):
            post_url (str | Unset):
            published_at (datetime.datetime | Unset):
     """

    created_at: datetime.datetime
    status: str
    title: str
    audience: str | Unset = UNSET
    campaign_id: str | Unset = UNSET
    notified: str | Unset = UNSET
    post_url: str | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        status = self.status

        title = self.title

        audience = self.audience

        campaign_id = self.campaign_id

        notified = self.notified

        post_url = self.post_url

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "status": status,
            "title": title,
        })
        if audience is not UNSET:
            field_dict["audience"] = audience
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if notified is not UNSET:
            field_dict["notified"] = notified
        if post_url is not UNSET:
            field_dict["postUrl"] = post_url
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        status = d.pop("status")

        title = d.pop("title")

        audience = d.pop("audience", UNSET)

        campaign_id = d.pop("campaignId", UNSET)

        notified = d.pop("notified", UNSET)

        post_url = d.pop("postUrl", UNSET)

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        integration_site_draft = cls(
            created_at=created_at,
            status=status,
            title=title,
            audience=audience,
            campaign_id=campaign_id,
            notified=notified,
            post_url=post_url,
            published_at=published_at,
        )

        return integration_site_draft


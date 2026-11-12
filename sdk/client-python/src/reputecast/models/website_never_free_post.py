from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WebsiteNeverFreePost")



@_attrs_define
class WebsiteNeverFreePost:
    """ 
        Attributes:
            id (str):
            status (str):
            tiers (list[str]):
            title (str):
            visibility (str):
            publish_at (datetime.datetime | Unset):
            published_at (datetime.datetime | Unset):
     """

    id: str
    status: str
    tiers: list[str]
    title: str
    visibility: str
    publish_at: datetime.datetime | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        status = self.status

        tiers = self.tiers



        title = self.title

        visibility = self.visibility

        publish_at: str | Unset = UNSET
        if not isinstance(self.publish_at, Unset):
            publish_at = self.publish_at.isoformat()

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "status": status,
            "tiers": tiers,
            "title": title,
            "visibility": visibility,
        })
        if publish_at is not UNSET:
            field_dict["publishAt"] = publish_at
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        status = d.pop("status")

        tiers = cast(list[str], d.pop("tiers"))


        title = d.pop("title")

        visibility = d.pop("visibility")

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




        website_never_free_post = cls(
            id=id,
            status=status,
            tiers=tiers,
            title=title,
            visibility=visibility,
            publish_at=publish_at,
            published_at=published_at,
        )

        return website_never_free_post


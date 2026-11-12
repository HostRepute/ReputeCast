from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SearchHit")



@_attrs_define
class SearchHit:
    """ 
        Attributes:
            id (str):
            title (str):
            at (datetime.datetime | Unset): A campaign's start, or its scheduled time
            channel (str | Unset):
            count (int | Unset):
            detail (str | Unset):
            slug (str | Unset):
            status (str | Unset):
     """

    id: str
    title: str
    at: datetime.datetime | Unset = UNSET
    channel: str | Unset = UNSET
    count: int | Unset = UNSET
    detail: str | Unset = UNSET
    slug: str | Unset = UNSET
    status: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        title = self.title

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        channel = self.channel

        count = self.count

        detail = self.detail

        slug = self.slug

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "title": title,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if channel is not UNSET:
            field_dict["channel"] = channel
        if count is not UNSET:
            field_dict["count"] = count
        if detail is not UNSET:
            field_dict["detail"] = detail
        if slug is not UNSET:
            field_dict["slug"] = slug
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        title = d.pop("title")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        channel = d.pop("channel", UNSET)

        count = d.pop("count", UNSET)

        detail = d.pop("detail", UNSET)

        slug = d.pop("slug", UNSET)

        status = d.pop("status", UNSET)

        search_hit = cls(
            id=id,
            title=title,
            at=at,
            channel=channel,
            count=count,
            detail=detail,
            slug=slug,
            status=status,
        )

        return search_hit


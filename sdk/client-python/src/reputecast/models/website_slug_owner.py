from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WebsiteSlugOwner")



@_attrs_define
class WebsiteSlugOwner:
    """ 
        Attributes:
            id (str):
            since (datetime.datetime):
            status (str):
            title (str):
     """

    id: str
    since: datetime.datetime
    status: str
    title: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        since = self.since.isoformat()

        status = self.status

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "since": since,
            "status": status,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        status = d.pop("status")

        title = d.pop("title")

        website_slug_owner = cls(
            id=id,
            since=since,
            status=status,
            title=title,
        )

        return website_slug_owner


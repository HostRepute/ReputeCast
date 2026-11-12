from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="DigestFeedHop")



@_attrs_define
class DigestFeedHop:
    """ 
        Attributes:
            status (int):
            url (str):
            location (str | Unset):
     """

    status: int
    url: str
    location: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        status = self.status

        url = self.url

        location = self.location


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
            "url": url,
        })
        if location is not UNSET:
            field_dict["location"] = location

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = d.pop("status")

        url = d.pop("url")

        location = d.pop("location", UNSET)

        digest_feed_hop = cls(
            status=status,
            url=url,
            location=location,
        )

        return digest_feed_hop


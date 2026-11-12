from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushReviewSite")



@_attrs_define
class PushReviewSite:
    """ 
        Attributes:
            host (str):
            subscribers (int):
     """

    host: str
    subscribers: int





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        subscribers = self.subscribers


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "subscribers": subscribers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        subscribers = d.pop("subscribers")

        push_review_site = cls(
            host=host,
            subscribers=subscribers,
        )

        return push_review_site


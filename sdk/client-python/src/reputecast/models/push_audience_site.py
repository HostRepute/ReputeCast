from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushAudienceSite")



@_attrs_define
class PushAudienceSite:
    """ 
        Attributes:
            host (str):
            id (str):
            selected (bool):
            subscribers (int):
     """

    host: str
    id: str
    selected: bool
    subscribers: int





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        id = self.id

        selected = self.selected

        subscribers = self.subscribers


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "id": id,
            "selected": selected,
            "subscribers": subscribers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        id = d.pop("id")

        selected = d.pop("selected")

        subscribers = d.pop("subscribers")

        push_audience_site = cls(
            host=host,
            id=id,
            selected=selected,
            subscribers=subscribers,
        )

        return push_audience_site


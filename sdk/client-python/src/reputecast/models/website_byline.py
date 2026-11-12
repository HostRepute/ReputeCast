from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsiteByline")



@_attrs_define
class WebsiteByline:
    """ 
        Attributes:
            name (str):
            color (str | Unset):
            user_id (str | Unset):
     """

    name: str
    color: str | Unset = UNSET
    user_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        color = self.color

        user_id = self.user_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if color is not UNSET:
            field_dict["color"] = color
        if user_id is not UNSET:
            field_dict["userId"] = user_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        color = d.pop("color", UNSET)

        user_id = d.pop("userId", UNSET)

        website_byline = cls(
            name=name,
            color=color,
            user_id=user_id,
        )

        return website_byline


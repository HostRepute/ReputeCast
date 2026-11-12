from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="APIKeyCreator")



@_attrs_define
class APIKeyCreator:
    """ 
        Attributes:
            avatar_color (str):
            name (str):
            id (str | Unset):
     """

    avatar_color: str
    name: str
    id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        avatar_color = self.avatar_color

        name = self.name

        id = self.id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "avatarColor": avatar_color,
            "name": name,
        })
        if id is not UNSET:
            field_dict["id"] = id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        avatar_color = d.pop("avatarColor")

        name = d.pop("name")

        id = d.pop("id", UNSET)

        api_key_creator = cls(
            avatar_color=avatar_color,
            name=name,
            id=id,
        )

        return api_key_creator


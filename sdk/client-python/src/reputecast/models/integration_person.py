from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="IntegrationPerson")



@_attrs_define
class IntegrationPerson:
    """ 
        Attributes:
            name (str):
            avatar_color (str | Unset):
            id (str | Unset):
     """

    name: str
    avatar_color: str | Unset = UNSET
    id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        avatar_color = self.avatar_color

        id = self.id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if avatar_color is not UNSET:
            field_dict["avatarColor"] = avatar_color
        if id is not UNSET:
            field_dict["id"] = id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        avatar_color = d.pop("avatarColor", UNSET)

        id = d.pop("id", UNSET)

        integration_person = cls(
            name=name,
            avatar_color=avatar_color,
            id=id,
        )

        return integration_person


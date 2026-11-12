from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AgencyPersonRef")



@_attrs_define
class AgencyPersonRef:
    """ 
        Attributes:
            avatar_color (str):
            email (str):
            id (str):
            name (str):
     """

    avatar_color: str
    email: str
    id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        avatar_color = self.avatar_color

        email = self.email

        id = self.id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "avatarColor": avatar_color,
            "email": email,
            "id": id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        avatar_color = d.pop("avatarColor")

        email = d.pop("email")

        id = d.pop("id")

        name = d.pop("name")

        agency_person_ref = cls(
            avatar_color=avatar_color,
            email=email,
            id=id,
            name=name,
        )

        return agency_person_ref


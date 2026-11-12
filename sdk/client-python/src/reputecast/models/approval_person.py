from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ApprovalPerson")



@_attrs_define
class ApprovalPerson:
    """ 
        Attributes:
            color (str):
            email (str):
            id (str):
            initials (str):
            name (str):
            role (str):
            presence (str | Unset):
     """

    color: str
    email: str
    id: str
    initials: str
    name: str
    role: str
    presence: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        color = self.color

        email = self.email

        id = self.id

        initials = self.initials

        name = self.name

        role = self.role

        presence = self.presence


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "email": email,
            "id": id,
            "initials": initials,
            "name": name,
            "role": role,
        })
        if presence is not UNSET:
            field_dict["presence"] = presence

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        color = d.pop("color")

        email = d.pop("email")

        id = d.pop("id")

        initials = d.pop("initials")

        name = d.pop("name")

        role = d.pop("role")

        presence = d.pop("presence", UNSET)

        approval_person = cls(
            color=color,
            email=email,
            id=id,
            initials=initials,
            name=name,
            role=role,
            presence=presence,
        )

        return approval_person


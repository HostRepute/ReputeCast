from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DigestPerson")



@_attrs_define
class DigestPerson:
    """ 
        Attributes:
            color (str):
            id (str):
            initials (str):
            name (str):
     """

    color: str
    id: str
    initials: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        color = self.color

        id = self.id

        initials = self.initials

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "id": id,
            "initials": initials,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        color = d.pop("color")

        id = d.pop("id")

        initials = d.pop("initials")

        name = d.pop("name")

        digest_person = cls(
            color=color,
            id=id,
            initials=initials,
            name=name,
        )

        return digest_person


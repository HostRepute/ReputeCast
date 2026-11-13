from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReplyPerson")



@_attrs_define
class ReplyPerson:
    """ 
        Attributes:
            email (str):
            name (str):
     """

    email: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        name = d.pop("name")

        reply_person = cls(
            email=email,
            name=name,
        )

        return reply_person


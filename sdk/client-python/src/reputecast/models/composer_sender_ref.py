from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ComposerSenderRef")



@_attrs_define
class ComposerSenderRef:
    """ 
        Attributes:
            email (str):
            id (str):
            name (str):
     """

    email: str
    id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        id = self.id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "id": id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        id = d.pop("id")

        name = d.pop("name")

        composer_sender_ref = cls(
            email=email,
            id=id,
            name=name,
        )

        return composer_sender_ref


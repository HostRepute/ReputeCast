from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReplyContact")



@_attrs_define
class ReplyContact:
    """ 
        Attributes:
            email (str):
            id (str):
            name (str):
            status (str):
     """

    email: str
    id: str
    name: str
    status: str





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        id = self.id

        name = self.name

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "id": id,
            "name": name,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        id = d.pop("id")

        name = d.pop("name")

        status = d.pop("status")

        reply_contact = cls(
            email=email,
            id=id,
            name=name,
            status=status,
        )

        return reply_contact


from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AuditActorRef")



@_attrs_define
class AuditActorRef:
    """ 
        Attributes:
            id (str):
            name (str):
            type_ (str):
     """

    id: str
    name: str
    type_: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        type_ = d.pop("type")

        audit_actor_ref = cls(
            id=id,
            name=name,
            type_=type_,
        )

        return audit_actor_ref


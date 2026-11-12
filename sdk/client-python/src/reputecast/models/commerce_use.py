from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_use_kind import CommerceUseKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="CommerceUse")



@_attrs_define
class CommerceUse:
    """ 
        Attributes:
            kind (CommerceUseKind):
            id (str | Unset):
            name (str | Unset):
            status (str | Unset):
     """

    kind: CommerceUseKind
    id: str | Unset = UNSET
    name: str | Unset = UNSET
    status: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        id = self.id

        name = self.name

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if id is not UNSET:
            field_dict["id"] = id
        if name is not UNSET:
            field_dict["name"] = name
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = CommerceUseKind(d.pop("kind"))




        id = d.pop("id", UNSET)

        name = d.pop("name", UNSET)

        status = d.pop("status", UNSET)

        commerce_use = cls(
            kind=kind,
            id=id,
            name=name,
            status=status,
        )

        return commerce_use


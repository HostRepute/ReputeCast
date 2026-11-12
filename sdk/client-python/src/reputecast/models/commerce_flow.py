from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="CommerceFlow")



@_attrs_define
class CommerceFlow:
    """ 
        Attributes:
            id (str):
            kind (str):
            name (str):
            status (str):
            pause_reason (str | Unset):
     """

    id: str
    kind: str
    name: str
    status: str
    pause_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind

        name = self.name

        status = self.status

        pause_reason = self.pause_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "kind": kind,
            "name": name,
            "status": status,
        })
        if pause_reason is not UNSET:
            field_dict["pauseReason"] = pause_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = d.pop("kind")

        name = d.pop("name")

        status = d.pop("status")

        pause_reason = d.pop("pauseReason", UNSET)

        commerce_flow = cls(
            id=id,
            kind=kind,
            name=name,
            status=status,
            pause_reason=pause_reason,
        )

        return commerce_flow


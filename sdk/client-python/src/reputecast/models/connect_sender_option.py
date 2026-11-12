from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ConnectSenderOption")



@_attrs_define
class ConnectSenderOption:
    """ 
        Attributes:
            id (str):
            label (str):
            verified (bool):
     """

    id: str
    label: str
    verified: bool





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        label = self.label

        verified = self.verified


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "label": label,
            "verified": verified,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        verified = d.pop("verified")

        connect_sender_option = cls(
            id=id,
            label=label,
            verified=verified,
        )

        return connect_sender_option


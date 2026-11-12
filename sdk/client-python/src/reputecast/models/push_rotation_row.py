from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushRotationRow")



@_attrs_define
class PushRotationRow:
    """ 
        Attributes:
            host (str):
            subscriptions (int):
     """

    host: str
    subscriptions: int





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        subscriptions = self.subscriptions


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "subscriptions": subscriptions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        subscriptions = d.pop("subscriptions")

        push_rotation_row = cls(
            host=host,
            subscriptions=subscriptions,
        )

        return push_rotation_row


from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="APIKeyLockedPermission")



@_attrs_define
class APIKeyLockedPermission:
    """ 
        Attributes:
            key (str):
            reason (str):
     """

    key: str
    reason: str





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        reason = d.pop("reason")

        api_key_locked_permission = cls(
            key=key,
            reason=reason,
        )

        return api_key_locked_permission


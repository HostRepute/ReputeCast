from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ContactSuppressInput")



@_attrs_define
class ContactSuppressInput:
    """ 
        Attributes:
            reason (str): Why, for the audit log and the list
     """

    reason: str





    def to_dict(self) -> dict[str, Any]:
        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        reason = d.pop("reason")

        contact_suppress_input = cls(
            reason=reason,
        )

        return contact_suppress_input


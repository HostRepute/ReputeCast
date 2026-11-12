from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ApprovalWhatsAppValue")



@_attrs_define
class ApprovalWhatsAppValue:
    """ 
        Attributes:
            key (str):
            source (str):
            fallback (str | Unset):
     """

    key: str
    source: str
    fallback: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        source = self.source

        fallback = self.fallback


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "source": source,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        source = d.pop("source")

        fallback = d.pop("fallback", UNSET)

        approval_whats_app_value = cls(
            key=key,
            source=source,
            fallback=fallback,
        )

        return approval_whats_app_value


from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AuditSource")



@_attrs_define
class AuditSource:
    """ 
        Attributes:
            channel (str):
            detail (str | Unset):
     """

    channel: str
    detail: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        channel = self.channel

        detail = self.detail


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "channel": channel,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        channel = d.pop("channel")

        detail = d.pop("detail", UNSET)

        audit_source = cls(
            channel=channel,
            detail=detail,
        )

        return audit_source


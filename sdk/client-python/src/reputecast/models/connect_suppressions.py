from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectSuppressions")



@_attrs_define
class ConnectSuppressions:
    """ 
        Attributes:
            available (bool):
            bounces (int):
            complaints (int):
            provider (str):
            short (str):
            total (int):
            error (str | Unset):
     """

    available: bool
    bounces: int
    complaints: int
    provider: str
    short: str
    total: int
    error: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        bounces = self.bounces

        complaints = self.complaints

        provider = self.provider

        short = self.short

        total = self.total

        error = self.error


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "bounces": bounces,
            "complaints": complaints,
            "provider": provider,
            "short": short,
            "total": total,
        })
        if error is not UNSET:
            field_dict["error"] = error

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        bounces = d.pop("bounces")

        complaints = d.pop("complaints")

        provider = d.pop("provider")

        short = d.pop("short")

        total = d.pop("total")

        error = d.pop("error", UNSET)

        connect_suppressions = cls(
            available=available,
            bounces=bounces,
            complaints=complaints,
            provider=provider,
            short=short,
            total=total,
            error=error,
        )

        return connect_suppressions


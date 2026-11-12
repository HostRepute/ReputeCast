from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SunsetDryRun")



@_attrs_define
class SunsetDryRun:
    """ 
        Attributes:
            erase (int):
            from_ (str): The first run's day (2026-12-01), in the workspace's time zone
            kept (int): Paid members, kept until their subscription ends
     """

    erase: int
    from_: str
    kept: int





    def to_dict(self) -> dict[str, Any]:
        erase = self.erase

        from_ = self.from_

        kept = self.kept


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "erase": erase,
            "from": from_,
            "kept": kept,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        erase = d.pop("erase")

        from_ = d.pop("from")

        kept = d.pop("kept")

        sunset_dry_run = cls(
            erase=erase,
            from_=from_,
            kept=kept,
        )

        return sunset_dry_run


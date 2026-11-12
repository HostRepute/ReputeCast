from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PollAutomatedVisits")



@_attrs_define
class PollAutomatedVisits:
    """ 
        Attributes:
            gateways (int):
            prefetch (int):
            total (int):
     """

    gateways: int
    prefetch: int
    total: int





    def to_dict(self) -> dict[str, Any]:
        gateways = self.gateways

        prefetch = self.prefetch

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "gateways": gateways,
            "prefetch": prefetch,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        gateways = d.pop("gateways")

        prefetch = d.pop("prefetch")

        total = d.pop("total")

        poll_automated_visits = cls(
            gateways=gateways,
            prefetch=prefetch,
            total=total,
        )

        return poll_automated_visits


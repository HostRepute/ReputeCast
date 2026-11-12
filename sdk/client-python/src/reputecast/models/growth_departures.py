from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GrowthDepartures")



@_attrs_define
class GrowthDepartures:
    """ 
        Attributes:
            complained (int):
            erased (int):
            hard_bounced (int):
            unsubscribed (int):
     """

    complained: int
    erased: int
    hard_bounced: int
    unsubscribed: int





    def to_dict(self) -> dict[str, Any]:
        complained = self.complained

        erased = self.erased

        hard_bounced = self.hard_bounced

        unsubscribed = self.unsubscribed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "complained": complained,
            "erased": erased,
            "hardBounced": hard_bounced,
            "unsubscribed": unsubscribed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        complained = d.pop("complained")

        erased = d.pop("erased")

        hard_bounced = d.pop("hardBounced")

        unsubscribed = d.pop("unsubscribed")

        growth_departures = cls(
            complained=complained,
            erased=erased,
            hard_bounced=hard_bounced,
            unsubscribed=unsubscribed,
        )

        return growth_departures


from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiveExcluded")



@_attrs_define
class LiveExcluded:
    """ 
        Attributes:
            bounced (int):
            complained (int):
            duplicates (int):
            excluded (int):
            paused (int):
            role (int):
            sunset (int):
            suppressed (int):
            tiers (int):
            unsubscribed (int):
     """

    bounced: int
    complained: int
    duplicates: int
    excluded: int
    paused: int
    role: int
    sunset: int
    suppressed: int
    tiers: int
    unsubscribed: int





    def to_dict(self) -> dict[str, Any]:
        bounced = self.bounced

        complained = self.complained

        duplicates = self.duplicates

        excluded = self.excluded

        paused = self.paused

        role = self.role

        sunset = self.sunset

        suppressed = self.suppressed

        tiers = self.tiers

        unsubscribed = self.unsubscribed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounced": bounced,
            "complained": complained,
            "duplicates": duplicates,
            "excluded": excluded,
            "paused": paused,
            "role": role,
            "sunset": sunset,
            "suppressed": suppressed,
            "tiers": tiers,
            "unsubscribed": unsubscribed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounced = d.pop("bounced")

        complained = d.pop("complained")

        duplicates = d.pop("duplicates")

        excluded = d.pop("excluded")

        paused = d.pop("paused")

        role = d.pop("role")

        sunset = d.pop("sunset")

        suppressed = d.pop("suppressed")

        tiers = d.pop("tiers")

        unsubscribed = d.pop("unsubscribed")

        live_excluded = cls(
            bounced=bounced,
            complained=complained,
            duplicates=duplicates,
            excluded=excluded,
            paused=paused,
            role=role,
            sunset=sunset,
            suppressed=suppressed,
            tiers=tiers,
            unsubscribed=unsubscribed,
        )

        return live_excluded


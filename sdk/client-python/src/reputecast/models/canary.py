from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.canary_watch_minutes import CanaryWatchMinutes
from typing import cast

if TYPE_CHECKING:
  from ..models.canary_gates import CanaryGates





T = TypeVar("T", bound="Canary")



@_attrs_define
class Canary:
    """ 
        Attributes:
            enabled (bool):
            gates (CanaryGates):
            never_mailed (bool):
            new_domain (bool):
            over_usual (bool):
            percent (float):
            seeds (bool):
            watch_minutes (CanaryWatchMinutes):
     """

    enabled: bool
    gates: CanaryGates
    never_mailed: bool
    new_domain: bool
    over_usual: bool
    percent: float
    seeds: bool
    watch_minutes: CanaryWatchMinutes





    def to_dict(self) -> dict[str, Any]:
        from ..models.canary_gates import CanaryGates # noqa: PLC0415
        enabled = self.enabled

        gates = self.gates.to_dict()

        never_mailed = self.never_mailed

        new_domain = self.new_domain

        over_usual = self.over_usual

        percent = self.percent

        seeds = self.seeds

        watch_minutes = self.watch_minutes.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
            "gates": gates,
            "neverMailed": never_mailed,
            "newDomain": new_domain,
            "overUsual": over_usual,
            "percent": percent,
            "seeds": seeds,
            "watchMinutes": watch_minutes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.canary_gates import CanaryGates # noqa: PLC0415
        d = dict(src_dict)
        enabled = d.pop("enabled")

        gates = CanaryGates.from_dict(d.pop("gates"))




        never_mailed = d.pop("neverMailed")

        new_domain = d.pop("newDomain")

        over_usual = d.pop("overUsual")

        percent = d.pop("percent")

        seeds = d.pop("seeds")

        watch_minutes = CanaryWatchMinutes(d.pop("watchMinutes"))




        canary = cls(
            enabled=enabled,
            gates=gates,
            never_mailed=never_mailed,
            new_domain=new_domain,
            over_usual=over_usual,
            percent=percent,
            seeds=seeds,
            watch_minutes=watch_minutes,
        )

        return canary


from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="HoldoutSettings")



@_attrs_define
class HoldoutSettings:
    """ 
        Attributes:
            automations (bool):
            basis_points (int):
            campaigns (bool):
            held_out (int):
            measured (bool):
            on (bool):
            percent (float):
            started_at (datetime.datetime | Unset):
     """

    automations: bool
    basis_points: int
    campaigns: bool
    held_out: int
    measured: bool
    on: bool
    percent: float
    started_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        automations = self.automations

        basis_points = self.basis_points

        campaigns = self.campaigns

        held_out = self.held_out

        measured = self.measured

        on = self.on

        percent = self.percent

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automations": automations,
            "basisPoints": basis_points,
            "campaigns": campaigns,
            "heldOut": held_out,
            "measured": measured,
            "on": on,
            "percent": percent,
        })
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        automations = d.pop("automations")

        basis_points = d.pop("basisPoints")

        campaigns = d.pop("campaigns")

        held_out = d.pop("heldOut")

        measured = d.pop("measured")

        on = d.pop("on")

        percent = d.pop("percent")

        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        holdout_settings = cls(
            automations=automations,
            basis_points=basis_points,
            campaigns=campaigns,
            held_out=held_out,
            measured=measured,
            on=on,
            percent=percent,
            started_at=started_at,
        )

        return holdout_settings


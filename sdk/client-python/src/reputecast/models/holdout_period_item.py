from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="HoldoutPeriodItem")



@_attrs_define
class HoldoutPeriodItem:
    """ 
        Attributes:
            automations (bool):
            basis_points (int):
            campaigns (bool):
            id (str):
            open_ (bool):
            percent (float):
            started_at (datetime.datetime):
            ended_at (datetime.datetime | Unset):
     """

    automations: bool
    basis_points: int
    campaigns: bool
    id: str
    open_: bool
    percent: float
    started_at: datetime.datetime
    ended_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        automations = self.automations

        basis_points = self.basis_points

        campaigns = self.campaigns

        id = self.id

        open_ = self.open_

        percent = self.percent

        started_at = self.started_at.isoformat()

        ended_at: str | Unset = UNSET
        if not isinstance(self.ended_at, Unset):
            ended_at = self.ended_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automations": automations,
            "basisPoints": basis_points,
            "campaigns": campaigns,
            "id": id,
            "open": open_,
            "percent": percent,
            "startedAt": started_at,
        })
        if ended_at is not UNSET:
            field_dict["endedAt"] = ended_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        automations = d.pop("automations")

        basis_points = d.pop("basisPoints")

        campaigns = d.pop("campaigns")

        id = d.pop("id")

        open_ = d.pop("open")

        percent = d.pop("percent")

        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        _ended_at = d.pop("endedAt", UNSET)
        ended_at: datetime.datetime | Unset
        if isinstance(_ended_at,  Unset):
            ended_at = UNSET
        else:
            ended_at = datetime.datetime.fromisoformat(_ended_at)




        holdout_period_item = cls(
            automations=automations,
            basis_points=basis_points,
            campaigns=campaigns,
            id=id,
            open_=open_,
            percent=percent,
            started_at=started_at,
            ended_at=ended_at,
        )

        return holdout_period_item


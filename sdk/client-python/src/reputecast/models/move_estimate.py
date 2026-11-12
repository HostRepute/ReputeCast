from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.move_check import MoveCheck





T = TypeVar("T", bound="MoveEstimate")



@_attrs_define
class MoveEstimate:
    """ 
        Attributes:
            extra_cost (float):
            extra_seconds (int):
            rate (float):
            risk (str):
            risk_text (str):
            target (str):
            target_id (str):
            target_short (str):
            checks (list[MoveCheck] | Unset):
            quota_free (int | Unset):
            quota_max (int | Unset):
     """

    extra_cost: float
    extra_seconds: int
    rate: float
    risk: str
    risk_text: str
    target: str
    target_id: str
    target_short: str
    checks: list[MoveCheck] | Unset = UNSET
    quota_free: int | Unset = UNSET
    quota_max: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.move_check import MoveCheck # noqa: PLC0415
        extra_cost = self.extra_cost

        extra_seconds = self.extra_seconds

        rate = self.rate

        risk = self.risk

        risk_text = self.risk_text

        target = self.target

        target_id = self.target_id

        target_short = self.target_short

        checks: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.checks, Unset):
            checks = []
            for checks_item_data in self.checks:
                checks_item = checks_item_data.to_dict()
                checks.append(checks_item)



        quota_free = self.quota_free

        quota_max = self.quota_max


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "extraCost": extra_cost,
            "extraSeconds": extra_seconds,
            "rate": rate,
            "risk": risk,
            "riskText": risk_text,
            "target": target,
            "targetId": target_id,
            "targetShort": target_short,
        })
        if checks is not UNSET:
            field_dict["checks"] = checks
        if quota_free is not UNSET:
            field_dict["quotaFree"] = quota_free
        if quota_max is not UNSET:
            field_dict["quotaMax"] = quota_max

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.move_check import MoveCheck # noqa: PLC0415
        d = dict(src_dict)
        extra_cost = d.pop("extraCost")

        extra_seconds = d.pop("extraSeconds")

        rate = d.pop("rate")

        risk = d.pop("risk")

        risk_text = d.pop("riskText")

        target = d.pop("target")

        target_id = d.pop("targetId")

        target_short = d.pop("targetShort")

        _checks = d.pop("checks", UNSET)
        checks: list[MoveCheck] | Unset = UNSET
        if _checks is not UNSET:
            checks = []
            for checks_item_data in _checks:
                checks_item = MoveCheck.from_dict(checks_item_data)



                checks.append(checks_item)


        quota_free = d.pop("quotaFree", UNSET)

        quota_max = d.pop("quotaMax", UNSET)

        move_estimate = cls(
            extra_cost=extra_cost,
            extra_seconds=extra_seconds,
            rate=rate,
            risk=risk,
            risk_text=risk_text,
            target=target,
            target_id=target_id,
            target_short=target_short,
            checks=checks,
            quota_free=quota_free,
            quota_max=quota_max,
        )

        return move_estimate


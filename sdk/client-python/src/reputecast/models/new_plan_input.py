from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.new_plan_input_days import NewPlanInputDays
from ..models.new_plan_input_guard_action import NewPlanInputGuardAction
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="NewPlanInput")



@_attrs_define
class NewPlanInput:
    """ 
        Attributes:
            days (NewPlanInputDays):
            engaged_first (bool):
            guard_action (NewPlanInputGuardAction):
            guard_bounce (float):
            guard_complaint (float):
            provider_id (UUID):
            spread_mailbox (bool):
            ip (str | Unset):
            overflow (bool | Unset):
            target_volume (int | Unset):
     """

    days: NewPlanInputDays
    engaged_first: bool
    guard_action: NewPlanInputGuardAction
    guard_bounce: float
    guard_complaint: float
    provider_id: UUID
    spread_mailbox: bool
    ip: str | Unset = UNSET
    overflow: bool | Unset = UNSET
    target_volume: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        days = self.days.value

        engaged_first = self.engaged_first

        guard_action = self.guard_action.value

        guard_bounce = self.guard_bounce

        guard_complaint = self.guard_complaint

        provider_id = str(self.provider_id)

        spread_mailbox = self.spread_mailbox

        ip = self.ip

        overflow = self.overflow

        target_volume = self.target_volume


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "engagedFirst": engaged_first,
            "guardAction": guard_action,
            "guardBounce": guard_bounce,
            "guardComplaint": guard_complaint,
            "providerId": provider_id,
            "spreadMailbox": spread_mailbox,
        })
        if ip is not UNSET:
            field_dict["ip"] = ip
        if overflow is not UNSET:
            field_dict["overflow"] = overflow
        if target_volume is not UNSET:
            field_dict["targetVolume"] = target_volume

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        days = NewPlanInputDays(d.pop("days"))




        engaged_first = d.pop("engagedFirst")

        guard_action = NewPlanInputGuardAction(d.pop("guardAction"))




        guard_bounce = d.pop("guardBounce")

        guard_complaint = d.pop("guardComplaint")

        provider_id = UUID(d.pop("providerId"))




        spread_mailbox = d.pop("spreadMailbox")

        ip = d.pop("ip", UNSET)

        overflow = d.pop("overflow", UNSET)

        target_volume = d.pop("targetVolume", UNSET)

        new_plan_input = cls(
            days=days,
            engaged_first=engaged_first,
            guard_action=guard_action,
            guard_bounce=guard_bounce,
            guard_complaint=guard_complaint,
            provider_id=provider_id,
            spread_mailbox=spread_mailbox,
            ip=ip,
            overflow=overflow,
            target_volume=target_volume,
        )

        return new_plan_input

